"use client";

import { MessageSquare, FileText, Upload, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DocumentList } from "./document-list";
import type { ChatSession } from "@/lib/models/cognitive.model";
import type { Document } from "@/lib/models/document.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";
import { Skeleton } from "@/components/ui/skeleton";
import { ui } from "@/lib/copy/ui";

export type KnowledgeMode = "chat" | "docs";

interface KnowledgeSidebarProps {
  mode: KnowledgeMode;
  onModeChange: (mode: KnowledgeMode) => void;
  sessions: ChatSession[];
  documents: Document[];
  currentSessionId: number | null;
  selectedDocId?: number | null;
  isSessionsLoading?: boolean;
  isDocumentsLoading?: boolean;
  isDocumentsFetching?: boolean;
  isDocumentsError?: boolean;
  onRetryDocuments?: () => void;
  onSelectSession: (sessionId: number) => void;
  onNewChat: () => void;
  onSelectDocument?: (doc: Document) => void;
  onUploadDocument: (file: File, title: string) => Promise<void>;
  onDeleteDocument: (documentId: number) => Promise<void>;
  onReprocessDocument?: (documentId: number) => Promise<void>;
  onRefreshDocuments: () => void;
  isUploading?: boolean;
  canManage?: boolean;
  onOpenUpload?: () => void;
}

export function KnowledgeSidebar({
  mode,
  onModeChange,
  sessions,
  documents,
  currentSessionId,
  selectedDocId = null,
  isSessionsLoading,
  isDocumentsLoading,
  isDocumentsFetching,
  isDocumentsError,
  onRetryDocuments,
  onSelectSession,
  onNewChat,
  onSelectDocument,
  onUploadDocument,
  onDeleteDocument,
  onReprocessDocument,
  onRefreshDocuments,
  isUploading,
  canManage = true,
  onOpenUpload,
}: KnowledgeSidebarProps) {
  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Segmented rail: Chat | Documentos */}
      <div className="p-3 border-b border-gray-200">
        <div className="flex rounded-lg p-1" style={{ backgroundColor: "#e5e7eb" }}>
          <button
            onClick={() => onModeChange("chat")}
            aria-pressed={mode === "chat"}
            className="flex-1 flex items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            style={{
              backgroundColor: mode === "chat" ? "white" : "transparent",
              color: mode === "chat" ? "#111827" : "#6b7280",
              boxShadow: mode === "chat" ? "0 1px 2px rgba(0,0,0,0.05)" : "none",
            }}
          >
            <MessageSquare className="h-3.5 w-3.5" />
            {ui.knowledge.modeChat}
          </button>
          <button
            onClick={() => onModeChange("docs")}
            aria-pressed={mode === "docs"}
            className="flex-1 flex items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            style={{
              backgroundColor: mode === "docs" ? "white" : "transparent",
              color: mode === "docs" ? "#111827" : "#6b7280",
              boxShadow: mode === "docs" ? "0 1px 2px rgba(0,0,0,0.05)" : "none",
            }}
          >
            <FileText className="h-3.5 w-3.5" />
            {ui.knowledge.modeDocs}
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {mode === "chat" ? (
          <ChatsTab
            sessions={sessions}
            currentSessionId={currentSessionId}
            isLoading={isSessionsLoading}
            onSelectSession={onSelectSession}
            onNewChat={onNewChat}
          />
        ) : (
          <DocsTab
            documents={documents}
            selectedDocId={selectedDocId}
            isLoading={isDocumentsLoading}
            isFetching={isDocumentsFetching}
            isError={isDocumentsError}
            onRetry={onRetryDocuments}
            onSelect={onSelectDocument}
            onUpload={onUploadDocument}
            onDelete={onDeleteDocument}
            onReprocess={onReprocessDocument}
            onRefresh={onRefreshDocuments}
            isUploading={isUploading}
            canManage={canManage}
          />
        )}
      </div>
    </div>
  );
}

function ChatsTab({
  sessions,
  currentSessionId,
  isLoading,
  onSelectSession,
  onNewChat,
}: {
  sessions: ChatSession[];
  currentSessionId: number | null;
  isLoading?: boolean;
  onSelectSession: (sessionId: number) => void;
  onNewChat: () => void;
}) {
  return (
    <>
      <div className="p-3">
        <Button onClick={onNewChat} variant="outline" size="sm" className="w-full gap-2">
          <Plus className="h-4 w-4" />
          {ui.knowledge.newChat}
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto px-3 pb-3">
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-12 w-full rounded-lg" />
            ))}
          </div>
        ) : sessions.length === 0 ? (
          <div className="text-center py-8">
            <MessageSquare className="mx-auto h-8 w-8 text-gray-300" />
            <p className="mt-2 text-sm text-gray-500">{ui.knowledge.newChat}</p>
          </div>
        ) : (
          <div className="space-y-1">
            {sessions.map((session) => {
              const isActive = currentSessionId === session.id;
              return (
                <button
                  key={session.id}
                  onClick={() => onSelectSession(session.id)}
                  className="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors"
                  style={{
                    backgroundColor: isActive ? "#ede9fe" : "transparent",
                    color: isActive ? "#5b21b6" : "#374151",
                  }}
                >
                  <p className="truncate font-medium">
                    {ChatHelpers.truncateTitle(session.title)}
                  </p>
                  <p className="text-xs mt-0.5" style={{ color: "#6b7280" }}>
                    {ChatHelpers.formatTimestamp(session.updatedAt)}
                  </p>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
}

function DocsTab({
  documents,
  selectedDocId,
  isLoading,
  isFetching,
  isError,
  onRetry,
  onSelect,
  onUpload,
  onDelete,
  onReprocess,
  onRefresh,
  isUploading,
  canManage,
  onOpenUpload,
}: {
  documents: Document[];
  selectedDocId?: number | null;
  isLoading?: boolean;
  isFetching?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  onSelect?: (doc: Document) => void;
  onUpload: (file: File, title: string) => Promise<void>;
  onDelete: (documentId: number) => Promise<void>;
  onReprocess?: (documentId: number) => Promise<void>;
  onRefresh: () => void;
  isUploading?: boolean;
  canManage?: boolean;
  onOpenUpload?: () => void;
}) {
  return (
    <>
      {canManage && (
        <div className="p-3">
          <Button
            variant="outline"
            size="sm"
            className="w-full gap-2"
            onClick={() => onOpenUpload?.()}
          >
            <Upload className="h-4 w-4" />
            {ui.knowledge.addDocument}
          </Button>
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-3 pb-3">
        <DocumentList
          documents={documents}
          isLoading={isLoading}
          isFetching={isFetching}
          isError={isError}
          onRetry={onRetry}
          onDelete={onDelete}
          onReprocess={onReprocess}
          onRefresh={onRefresh}
          onSelect={onSelect}
          selectedId={selectedDocId}
          compact={true}
          canManage={canManage}
        />
      </div>
    </>
  );
}
