"use client";

import { useState, useEffect } from "react";
import { useSessionsQuery, useSessionMessagesQuery } from "@/lib/hooks/queries/use-sessions-query";
import { useChat } from "@/lib/hooks/mutations/use-chat";
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
    refetch: refetchDocuments,
  } = useDocumentsQuery();

  const uploadMutation = useUploadDocument();
  const deleteMutation = useDeleteDocument();

  const documents = documentsData?.documents ?? [];
  const processedDocCount = documents.filter((d) => d.status === "processed").length;

  const handleUpload = async (file: File, title: string) => {
    await uploadMutation.mutateAsync({ file, title });
  };

  const handleDeleteDocument = async (documentId: number) => {
    await deleteMutation.mutateAsync({ documentId });
  };

  const [currentSessionId, setCurrentSessionId] = useState<number | null>(null);
  const [optimisticMessages, setOptimisticMessages] = useState<ChatMessage[]>([]);
  const [messageSources, setMessageSources] = useState<Record<number, SimilarDocument[]>>({});

  const {
    data: sessions,
    isLoading: isSessionsLoading,
  } = useSessionsQuery();

  const { data: sessionMessages, isLoading: isMessagesLoading } = useSessionMessagesQuery({
    sessionId: currentSessionId ?? 0,
    enabled: currentSessionId !== null && currentSessionId > 0,
  });

  const chatMutation = useChat();
  const messages = [...(sessionMessages ?? []), ...optimisticMessages];

  useEffect(() => {
    if (sessionMessages && sessionMessages.length > 0) {
      setOptimisticMessages([]);
    }
  }, [sessionMessages]);

  useEffect(() => {
    if (!currentSessionId && sessions && sessions.length > 0) {
      setCurrentSessionId(sessions[0].id);
    }
  }, [currentSessionId, sessions]);

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
    setOptimisticMessages((prev) => [...prev, optimisticUserMessage]);

    try {
      const response = await chatMutation.mutateAsync({
        sessionId: currentSessionId ?? undefined,
        message,
        useRag,
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
      setOptimisticMessages([]);
    } catch {
      setOptimisticMessages((prev) => prev.filter((m) => m.id !== optimisticUserMessage.id));
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
          isSending={chatMutation.isPending}
          onSendMessage={handleSendMessage}
          onNewChat={handleNewChat}
          messageSources={messageSources}
          documentCount={processedDocCount}
        />
      </div>
    </div>
  );
}
