"use client";

import { useState, useEffect } from "react";
import { useSessionsQuery, useSessionMessagesQuery } from "@/lib/hooks/queries/use-sessions-query";
import { useChatStream } from "@/lib/hooks/mutations/use-chat-stream";
import { useDocumentsQuery } from "@/lib/hooks/queries/use-documents-query";
import { useUploadDocument } from "@/lib/hooks/mutations/use-upload-document";
import { useDeleteDocument } from "@/lib/hooks/mutations/use-delete-document";
import { ChatInterface } from "./chat-interface";
import { KnowledgeSidebar } from "./knowledge-sidebar";
import type { ChatMessage, SimilarDocument } from "@/lib/models/cognitive.model";

export function KnowledgeContent() {
  const {
    data: documentsData,
    isLoading: isDocumentsLoading,
    isFetching: isDocumentsFetching,
    isError: isDocumentsError,
    refetch: refetchDocuments,
  } = useDocumentsQuery();

  const uploadMutation = useUploadDocument();
  const deleteMutation = useDeleteDocument();

  const documents = documentsData?.documents ?? [];
  const processedDocCount = documents.filter((d) => d.status === "processed").length;

  const handleUpload = async (file: File, title: string) => {
    try {
      await uploadMutation.mutateAsync({ file, title });
    } catch {
      // error handled by mutation
    }
  };

  const handleDeleteDocument = async (documentId: number) => {
    try {
      await deleteMutation.mutateAsync({ documentId });
    } catch {
      // error handled by mutation
    }
  };

  const [currentSessionId, setCurrentSessionId] = useState<number | null>(null);
  const [optimisticMessages, setOptimisticMessages] = useState<ChatMessage[]>([]);
  const [streamingMessageId, setStreamingMessageId] = useState<number | null>(null);
  const [messageSources, setMessageSources] = useState<Record<number, SimilarDocument[]>>({});

  const {
    data: sessions,
    isLoading: isSessionsLoading,
  } = useSessionsQuery();

  const { data: sessionMessages, isLoading: isMessagesLoading } = useSessionMessagesQuery({
    sessionId: currentSessionId ?? 0,
    enabled: currentSessionId !== null && currentSessionId > 0,
  });

  const { sendMessage: sendStreamMessage, isStreaming } = useChatStream();
  // Derive the visible list instead of mutating optimistic state on prop change:
  // hide optimistic messages that belong to another session or that the server
  // has already confirmed (same id).
  const serverMessages = sessionMessages ?? [];
  const visibleOptimisticMessages = optimisticMessages.filter(
    (m) =>
      m.sessionId === currentSessionId &&
      !serverMessages.some((s) => s.id === m.id)
  );
  const messages = [...serverMessages, ...visibleOptimisticMessages];

  // Select the first session automatically when sessions first load.
  const [prevSessions, setPrevSessions] = useState(sessions);
  if (sessions !== prevSessions) {
    setPrevSessions(sessions);
    if (!currentSessionId && sessions && sessions.length > 0) {
      setCurrentSessionId(sessions[0].id);
    }
  }

  const currentSession = sessions?.find((s) => s.id === currentSessionId);
  const sessionTitle = currentSession?.title;

  const handleSendMessage = async (message: string, useRag: boolean) => {
    const optimisticUserMessage: ChatMessage = {
      id: Date.now(),
      sessionId: currentSessionId ?? 0,
      role: "user",
      content: message,
      tokensUsed: 0,
      createdAt: new Date(),
    };
    const optimisticAssistantId = Date.now() + 1;
    setOptimisticMessages((prev) => [...prev, optimisticUserMessage]);
    setStreamingMessageId(optimisticAssistantId);

    try {
      const response = await sendStreamMessage({
        sessionId: currentSessionId ?? undefined,
        message,
        useRag,
      }, {
        onToken: (token) => {
          setOptimisticMessages((prev) => {
            const existing = prev.find((m) => m.id === optimisticAssistantId);
            if (existing) {
              return prev.map((m) =>
                m.id === optimisticAssistantId
                  ? { ...m, content: m.content + token }
                  : m
              );
            }
            return [
              ...prev,
              {
                id: optimisticAssistantId,
                sessionId: currentSessionId ?? 0,
                role: "assistant",
                content: token,
                tokensUsed: 0,
                createdAt: new Date(),
              },
            ];
          });
        },
        onDone: () => setStreamingMessageId(null),
      });

      if (response.sessionId && response.sessionId !== currentSessionId) {
        setCurrentSessionId(response.sessionId);
      }

      if (response.referencedDocs && response.referencedDocs.length > 0) {
        setMessageSources((prev) => ({
          ...prev,
          [response.message.id]: response.referencedDocs!,
        }));
      }
      setOptimisticMessages((prev) =>
        prev.filter((m) => m.id !== optimisticUserMessage.id)
      );
      setStreamingMessageId(null);
    } catch {
      setOptimisticMessages((prev) =>
        prev.filter((m) => m.id !== optimisticUserMessage.id && m.id !== optimisticAssistantId)
      );
      setStreamingMessageId(null);
    }
  };

  const handleNewChat = () => {
    setCurrentSessionId(null);
    setOptimisticMessages([]);
  };

  const handleSelectSession = (sessionId: number) => {
    setCurrentSessionId(sessionId);
    setOptimisticMessages([]);
  };

  return (
    <div className="flex h-[600px] rounded-lg border border-gray-200 bg-white overflow-hidden">
      {/* Fixed Sidebar */}
      <div className="w-64 border-r border-gray-200 flex-shrink-0 h-full overflow-hidden">
        <KnowledgeSidebar
          sessions={sessions ?? []}
          documents={documents}
          currentSessionId={currentSessionId}
          isSessionsLoading={isSessionsLoading}
          isDocumentsLoading={isDocumentsLoading}
          isDocumentsFetching={isDocumentsFetching}
          isDocumentsError={isDocumentsError}
          onRetryDocuments={() => refetchDocuments()}
          onSelectSession={handleSelectSession}
          onNewChat={handleNewChat}
          onUploadDocument={handleUpload}
          onDeleteDocument={handleDeleteDocument}
          onRefreshDocuments={() => refetchDocuments()}
          isUploading={uploadMutation.isPending}
        />
      </div>

      {/* Chat Area */}
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        <ChatInterface
          messages={messages}
          sessionTitle={sessionTitle}
          isLoading={isSessionsLoading || isMessagesLoading}
          isSending={isStreaming}
          streamingMessageId={streamingMessageId}
          onSendMessage={handleSendMessage}
          onNewChat={handleNewChat}
          messageSources={messageSources}
          documentCount={processedDocCount}
        />
      </div>
    </div>
  );
}
