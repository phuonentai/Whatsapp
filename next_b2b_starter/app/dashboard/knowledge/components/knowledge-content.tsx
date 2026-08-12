"use client";

import { useCallback, useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Sparkles, PanelRightClose, X, FileDown } from "lucide-react";
import { useSessionsQuery, useSessionMessagesQuery } from "@/lib/hooks/queries/use-sessions-query";
import { useChatStream } from "@/lib/hooks/mutations/use-chat-stream";
import { useDocumentsQuery } from "@/lib/hooks/queries/use-documents-query";
import { useUploadDocument } from "@/lib/hooks/mutations/use-upload-document";
import { useDeleteDocument } from "@/lib/hooks/mutations/use-delete-document";
import { useUpdateDocument } from "@/lib/hooks/mutations/use-update-document";
import { useReprocessDocument } from "@/lib/hooks/mutations/use-reprocess-document";
import { useAiUsageQuery } from "@/lib/hooks/queries/use-ai-usage-query";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import { markKnowledgeVisited } from "@/lib/onboarding/storage";
import { ChatInterface } from "./chat-interface";
import { KnowledgeSidebar, type KnowledgeMode } from "./knowledge-sidebar";
import { DocumentDetail, RestrictedDocumentState } from "./document-detail";
import { DocumentSources } from "./document-sources";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DocumentUpload } from "./document-upload";
import { cn } from "@/lib/utils";
import type { ChatMessage, SimilarDocument } from "@/lib/models/cognitive.model";
import type { DocumentVisibility } from "@/lib/models/document.model";
import { ui } from "@/lib/copy/ui";

const VALID_MODES = new Set(["chat", "docs"]);

// A minimal valid one-page PDF (built once) used by the guided empty state.
const SAMPLE_PDF_BASE64 =
  "JVBERi0xLjQKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFnZXMgMiAwIFIgPj4KZW5kb2JqCjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9LaWRzIFszIDAgUl0gL0NvdW50IDEgPj4KZW5kb2JqCjMgMCBvYmoKPDwgL1R5cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFCb3ggWzAgMCA2MTIgNzkyXSA+PgplbmRvYmoKeHJlZgowIDQKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTggMDAwMDAgbiAKMDAwMDAwMDExMSAwMDAwMCBuIAp0cmFpbGVyCjw8IC9Sb290IDEgMCBSIC9TaXplIDQgPj4Kc3RhcnR4cmVmCjE2NQolJUVPRgo=";

function downloadSamplePdf() {
  try {
    const binary = atob(SAMPLE_PDF_BASE64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    const blob = new Blob([bytes], { type: "application/pdf" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "muestra.pdf";
    a.click();
    URL.revokeObjectURL(url);
  } catch {
    // Download unavailable — the empty state copy still explains the flow.
  }
}

export function KnowledgeContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { hasPermission } = usePermissions();
  const canManage = hasPermission(PERMISSIONS.ORG_MANAGE);

  // Two-mode rail driven by the URL (?mode=chat|docs) so deep-links,
  // refresh and back/forward keep the state.
  const rawMode = searchParams.get("mode");
  const mode: KnowledgeMode = VALID_MODES.has(rawMode ?? "") ? (rawMode as KnowledgeMode) : "chat";
  const rawDoc = searchParams.get("doc");
  const docParamId = rawDoc ? Number(rawDoc) : null;

  const setMode = useCallback(
    (next: KnowledgeMode) => {
      const params = new URLSearchParams(searchParams.toString());
      if (next === "chat") params.delete("mode");
      else params.set("mode", next);
      const qs = params.toString();
      router.replace(qs ? `/dashboard/knowledge?${qs}` : "/dashboard/knowledge");
    },
    [router, searchParams]
  );

  const openDocument = useCallback(
    (documentId: number) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set("mode", "docs");
      params.set("doc", String(documentId));
      router.replace(`/dashboard/knowledge?${params.toString()}`);
    },
    [router, searchParams]
  );

  const [contextOpen, setContextOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);

  // Onboarding checklist: visiting the knowledge base completes the
  // "Agrega tu primer documento" step.
  useEffect(() => {
    markKnowledgeVisited();
  }, []);

  const {
    data: documentsData,
    isLoading: isDocumentsLoading,
    isFetching: isDocumentsFetching,
    isError: isDocumentsError,
    refetch: refetchDocuments,
  } = useDocumentsQuery();

  const { data: aiUsage } = useAiUsageQuery();
  const credits = aiUsage
    ? {
        used: aiUsage.credits_used,
        max: aiUsage.credits_max,
        remaining: aiUsage.credits_remaining,
      }
    : undefined;

  const uploadMutation = useUploadDocument();
  const deleteMutation = useDeleteDocument();
  const updateMutation = useUpdateDocument();
  const reprocessMutation = useReprocessDocument();

  const documents = documentsData?.documents ?? [];
  const indexingCount = documents.filter(
    (d) => d.status === "pending" || d.status === "processing"
  ).length;

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

  const handleRename = async (documentId: number, title: string) => {
    await updateMutation.mutateAsync({ documentId, fields: { title } });
  };

  const handleChangeVisibility = async (documentId: number, visibility: DocumentVisibility) => {
    await updateMutation.mutateAsync({ documentId, fields: { visibility } });
  };

  const handleReprocess = async (documentId: number) => {
    await reprocessMutation.mutateAsync({ documentId });
  };

  const [currentSessionId, setCurrentSessionId] = useState<number | null>(null);
  const [optimisticMessages, setOptimisticMessages] = useState<ChatMessage[]>([]);
  const [streamingMessageId, setStreamingMessageId] = useState<number | null>(null);
  const [messageSources, setMessageSources] = useState<Record<number, SimilarDocument[]>>({});
  const [aiBlocked, setAiBlocked] = useState(false);

  const {
    data: sessions,
    isLoading: isSessionsLoading,
  } = useSessionsQuery();

  const { data: sessionMessages, isLoading: isMessagesLoading } = useSessionMessagesQuery({
    sessionId: currentSessionId ?? 0,
    enabled: currentSessionId !== null && currentSessionId > 0,
  });

  const { sendMessage: sendStreamMessage, isStreaming } = useChatStream();
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
    } catch (err) {
      const code = (err as Error & { code?: string })?.code;
      if (code === "funcionalidad_no_disponible") {
        setAiBlocked(true);
      }
      setOptimisticMessages((prev) =>
        prev.filter((m) => m.id !== optimisticUserMessage.id && m.id !== optimisticAssistantId)
      );
      setStreamingMessageId(null);
    }
  };

  const handleNewChat = useCallback(() => {
    setCurrentSessionId(null);
    setOptimisticMessages([]);
  }, []);

  const aiNewChatSignal = useCommandPaletteStore((state) => state.aiNewChatSignal);
  useEffect(() => {
    if (aiNewChatSignal === 0) return;
    const timer = window.setTimeout(handleNewChat, 0);
    return () => window.clearTimeout(timer);
  }, [aiNewChatSignal, handleNewChat]);

  const handleSelectSession = (sessionId: number) => {
    setCurrentSessionId(sessionId);
    setOptimisticMessages([]);
  };

  // Docs mode: resolve the selected document. A doc id that is not in the
  // member's visible list (admin_only for a member, or deleted) renders the
  // restricted/not-available state without leaking the title.
  const selectedDocument =
    mode === "docs" && docParamId != null
      ? documents.find((d) => d.id === docParamId) ?? null
      : null;
  const selectedDocId = mode === "docs" ? selectedDocument?.id ?? null : null;

  // Context pane (xl+): sources of the last assistant message in chat mode;
  // document metadata in docs mode.
  const lastAssistantMessages = [...messages].reverse().find((m) => m.role === "assistant");
  const lastSources = lastAssistantMessages
    ? messageSources[lastAssistantMessages.id] ?? []
    : [];

  const lastSourceDoc = selectedDocument;

  // Upgrade gate: ai_assistant entitlement off (403 funcionalidad_no_disponible).
  if (aiBlocked) {
    return (
      <div className="flex h-[600px] items-center justify-center rounded-lg border border-gray-200 bg-white">
        <div className="max-w-md p-8 text-center">
          <div
            className="mx-auto flex h-12 w-12 items-center justify-center rounded-full"
            style={{ backgroundColor: "#ede9fe" }}
          >
            <Sparkles className="h-6 w-6" style={{ color: "#7c3aed" }} />
          </div>
          <h2 className="mt-4 text-base font-semibold" style={{ color: "#111827" }}>
            {ui.knowledge.upgradeTitle}
          </h2>
          <p className="mt-2 text-sm" style={{ color: "#6b7280" }}>
            {ui.knowledge.upgradeDesc}
          </p>
          <Link href="/dashboard/settings?view=subscription">
            <Button className="mt-6 bg-gray-900 hover:bg-gray-800">
              {ui.knowledge.upgradeCta}
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  const contextPane = mode === "docs" ? (
    lastSourceDoc ? (
      <div className="p-4">
        <p className="text-xs font-semibold uppercase tracking-wide" style={{ color: "#6b7280" }}>
          {ui.knowledge.modeDocs}
        </p>
        <div className="mt-3 space-y-3 text-sm">
          <div>
            <p className="text-xs" style={{ color: "#9ca3af" }}>Archivo</p>
            <p className="truncate font-medium" style={{ color: "#111827" }}>{lastSourceDoc.fileName}</p>
          </div>
          <div>
            <p className="text-xs" style={{ color: "#9ca3af" }}>Tamaño</p>
            <p style={{ color: "#111827" }}>{lastSourceDoc.fileSize} bytes</p>
          </div>
          <div>
            <p className="text-xs" style={{ color: "#9ca3af" }}>Estado</p>
            <p style={{ color: "#111827" }}>{lastSourceDoc.status}</p>
          </div>
          <div>
            <p className="text-xs" style={{ color: "#9ca3af" }}>Visibilidad</p>
            <p style={{ color: "#111827" }}>{lastSourceDoc.visibility}</p>
          </div>
        </div>
      </div>
    ) : null
  ) : lastSources.length > 0 ? (
    <div className="p-4">
      <p className="text-xs font-semibold uppercase tracking-wide" style={{ color: "#6b7280" }}>
        {ui.knowledge.modeChat}
      </p>
      <div className="mt-3">
        <DocumentSources sources={lastSources} onSourceClick={openDocument} />
      </div>
    </div>
  ) : null;

  return (
    <div className="flex h-[600px] rounded-lg border border-gray-200 bg-white overflow-hidden relative">
      {/* Rail */}
      <div className="w-64 border-r border-gray-200 flex-shrink-0 h-full overflow-hidden">
        <KnowledgeSidebar
          mode={mode}
          onModeChange={setMode}
          sessions={sessions ?? []}
          documents={documents}
          currentSessionId={currentSessionId}
          selectedDocId={selectedDocId}
          isSessionsLoading={isSessionsLoading}
          isDocumentsLoading={isDocumentsLoading}
          isDocumentsFetching={isDocumentsFetching}
          isDocumentsError={isDocumentsError}
          onRetryDocuments={() => refetchDocuments()}
          onSelectSession={handleSelectSession}
          onNewChat={handleNewChat}
          onSelectDocument={(doc) => openDocument(doc.id)}
          onUploadDocument={handleUpload}
          onDeleteDocument={handleDeleteDocument}
          onReprocessDocument={handleReprocess}
          onRefreshDocuments={() => refetchDocuments()}
          isUploading={uploadMutation.isPending}
          canManage={canManage}
          onOpenUpload={() => setUploadOpen(true)}
        />
      </div>

      {/* Main pane: thread ⇄ document detail */}
      <div className="flex-1 min-w-0 h-full overflow-hidden">
        {mode === "docs" ? (
          docParamId == null ? (
            <div className="flex h-full items-center justify-center p-8">
              <div className="max-w-md text-center">
                <h2 className="text-base font-semibold" style={{ color: "#111827" }}>
                  {ui.knowledge.emptyStateTitle}
                </h2>
                <p className="mt-2 text-sm" style={{ color: "#6b7280" }}>
                  {ui.knowledge.emptyStateDesc}
                </p>
                <button
                  onClick={downloadSamplePdf}
                  className="mt-5 inline-flex items-center gap-2 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-50"
                  style={{ color: "#374151" }}
                >
                  <FileDown className="h-4 w-4" />
                  {ui.knowledge.samplePdf}
                </button>
              </div>
            </div>
          ) : selectedDocument ? (
            <DocumentDetail
              document={selectedDocument}
              canManage={canManage}
              onRename={handleRename}
              onReprocess={handleReprocess}
              onDelete={handleDeleteDocument}
              onChangeVisibility={handleChangeVisibility}
              onBack={() => {
                const params = new URLSearchParams(searchParams.toString());
                params.delete("doc");
                router.replace(`/dashboard/knowledge?${params.toString()}`);
              }}
            />
          ) : (
            // Not in the member's visible list: admin_only for a member, or
            // deleted for an admin — always rendered without the title.
            <RestrictedDocumentState
              onBack={() => {
                const params = new URLSearchParams(searchParams.toString());
                params.delete("doc");
                router.replace(`/dashboard/knowledge?${params.toString()}`);
              }}
            />
          )
        ) : (
          <ChatInterface
            messages={messages}
            sessionTitle={sessionTitle}
            isLoading={isSessionsLoading || isMessagesLoading}
            isSending={isStreaming}
            streamingMessageId={streamingMessageId}
            onSendMessage={handleSendMessage}
            onNewChat={handleNewChat}
            messageSources={messageSources}
            onSourceClick={openDocument}
            credits={credits}
            indexingCount={indexingCount}
            canManage={canManage}
            onAddDocument={() => setUploadOpen(true)}
          />
        )}
      </div>

      {/* Context pane: xl+ static column; <1280px drawer; hidden on mobile */}
      <div
        className={cn(
          "w-80 border-l border-gray-200 bg-white flex-shrink-0 h-full",
          "hidden xl:block"
        )}
      >
        {contextPane}
      </div>

      {contextPane && (
        <button
          onClick={() => setContextOpen(!contextOpen)}
          className="absolute right-4 top-4 z-20 flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white shadow-sm xl:hidden"
          aria-label="Contexto"
        >
          {contextOpen ? <X className="h-4 w-4" /> : <PanelRightClose className="h-4 w-4" />}
        </button>
      )}

      {contextOpen && (
        <div className="absolute inset-y-0 right-0 z-10 w-80 border-l border-gray-200 bg-white shadow-lg sm:block xl:hidden overflow-y-auto">
          {contextPane}
        </div>
      )}

      {/* Shared upload dialog (header button in chat mode + rail button in docs
          mode both open the same dialog; upload is admin-only in v1). */}
      {canManage && (
        <Dialog open={uploadOpen} onOpenChange={setUploadOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{ui.knowledge.addDocument}</DialogTitle>
            </DialogHeader>
            <DocumentUpload onUpload={handleUpload} isUploading={uploadMutation.isPending} />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
