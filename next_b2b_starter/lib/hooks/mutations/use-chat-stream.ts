// lib/hooks/mutations/use-chat-stream.ts
// Streaming chat consumer. Sends the request to /example_cognitive/chat with
// Accept: text/event-stream and stream:true, parses the SSE token stream, and
// falls back to the single JSON response when streaming is unavailable.

import { useCallback, useState } from "react";
import { apiClient, resolveAccessToken } from "@/lib/api/api/client/api-client";
import type {
  ChatRequest,
  ChatResponse,
  SimilarDocument,
} from "@/lib/models/cognitive.model";

interface UseChatStreamOptions {
  onToken?: (token: string) => void;
  onDone?: (response: ChatResponse) => void;
}

interface StreamState {
  sessionId?: number;
  messageId?: number;
  fullText: string;
  tokensUsed: number;
  referencedDocs?: SimilarDocument[];
}

export function useChatStream(options: UseChatStreamOptions = {}) {
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendMessage = useCallback(
    async (request: ChatRequest, callOptions?: UseChatStreamOptions): Promise<ChatResponse> => {
      const { onToken = options.onToken, onDone = options.onDone } = callOptions ?? {};
      setIsStreaming(true);
      setError(null);
      const baseUrl = apiClient.getBaseUrl();

      try {
        const token = await resolveAccessToken();
        const headers: Record<string, string> = {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
        };
        if (token) headers["Authorization"] = `Bearer ${token}`;

        const controller = new AbortController();
        const response = await fetch(`${baseUrl}/example_cognitive/chat`, {
          method: "POST",
          headers,
          body: JSON.stringify({
            session_id: request.sessionId,
            message: request.message,
            use_rag: request.useRag ?? true,
            max_documents: request.maxDocuments,
            context_history: request.contextHistory,
            stream: true,
          }),
          signal: controller.signal,
        });

        if (!response.ok) {
          // Fall back to the JSON error body when available.
          let message = `Request failed (${response.status})`;
          let code: string | undefined;
          try {
            const body = await response.json();
            if (body?.message) message = body.message;
            // Surface the machine-readable code (e.g. funcionalidad_no_disponible)
            // so callers can render the entitlement upgrade gate.
            if (typeof body?.error === "string" && body.error) code = body.error;
          } catch {
            // non-JSON error body
          }
          const failed = new Error(message) as Error & { code?: string };
          if (code) failed.code = code;
          throw failed;
        }

        const contentType = response.headers.get("content-type") || "";
        if (!contentType.includes("text/event-stream")) {
          // Non-streaming fallback: parse the JSON response directly.
          const data = (await response.json()) as ChatResponse;
          onDone?.(data);
          return data;
        }

        // SSE stream parsing.
        const reader = response.body?.getReader();
        if (!reader) throw new Error("Response body is not readable");

        const decoder = new TextDecoder();
        let buffer = "";
        const state: StreamState = { fullText: "", tokensUsed: 0 };
        let doneEventReceived = false;

        const flushEvent = (dataLine: string) => {
          if (!dataLine.startsWith("data:")) return;
          const payload = dataLine.slice(5).trim();
          if (!payload) return;
          try {
            const parsed = JSON.parse(payload) as Record<string, unknown>;
            if (typeof parsed.token === "string") {
              state.fullText += parsed.token;
              onToken?.(parsed.token);
            } else if (parsed.done === true) {
              doneEventReceived = true;
              if (typeof parsed.session_id === "number") state.sessionId = parsed.session_id;
              if (typeof parsed.message_id === "number") state.messageId = parsed.message_id;
              if (Array.isArray(parsed.referenced_docs)) {
                state.referencedDocs = (parsed.referenced_docs as Array<Record<string, unknown>>)
                  .filter((d) => typeof d?.document_id === "number")
                  .map((d) => ({
                    id: typeof d.id === "number" ? d.id : 0,
                    documentId: d.document_id as number,
                    contentPreview:
                      typeof d.content_preview === "string" ? d.content_preview : "",
                    similarityScore:
                      typeof d.similarity_score === "number" ? d.similarity_score : 0,
                  }));
              }
            }
          } catch {
            // skip malformed event lines
          }
        };

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });

          // SSE frames are separated by a blank line.
          let frameEnd: number;
          while ((frameEnd = buffer.indexOf("\n\n")) !== -1) {
            const frame = buffer.slice(0, frameEnd);
            buffer = buffer.slice(frameEnd + 2);
            frame.split("\n").forEach(flushEvent);
          }
        }
        // Flush any trailing frame.
        buffer.split("\n").forEach(flushEvent);

        if (!doneEventReceived) {
          // Mid-stream failure or unsupported framing: fall back by parsing
          // whatever was accumulated as the assistant content.
          const fallback: ChatResponse = {
            sessionId: state.sessionId ?? 0,
            message: {
              id: state.messageId ?? 0,
              sessionId: state.sessionId ?? 0,
              role: "assistant",
              content: state.fullText,
              tokensUsed: 0,
              createdAt: new Date(),
            },
            tokensUsed: 0,
          };
          onDone?.(fallback);
          return fallback;
        }

        const chatResponse: ChatResponse = {
          sessionId: state.sessionId ?? 0,
          message: {
            id: state.messageId ?? 0,
            sessionId: state.sessionId ?? 0,
            role: "assistant",
            content: state.fullText,
            tokensUsed: 0,
            createdAt: new Date(),
          },
          referencedDocs: state.referencedDocs,
          tokensUsed: 0,
        };
        onDone?.(chatResponse);
        return chatResponse;
      } catch (e) {
        const message = e instanceof Error ? e.message : "Chat streaming failed";
        setError(message);
        throw e;
      } finally {
        setIsStreaming(false);
      }
    },
    [options]
  );

  return { sendMessage, isStreaming, error };
}
