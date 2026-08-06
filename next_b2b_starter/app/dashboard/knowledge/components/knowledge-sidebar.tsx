"use client";

import { useState } from "react";
import { MessageSquare, FileText, Upload, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { DocumentList } from "./document-list";
import { DocumentUpload } from "./document-upload";
import type { ChatSession } from "@/lib/models/cognitive.model";
import type { Document } from "@/lib/models/document.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";
import { Skeleton } from "@/components/ui/skeleton";

interface KnowledgeSidebarProps {
  sessions: ChatSession[];
  documents: Document[];
  currentSessionId: number | null;
  isSessionsLoading?: boolean;
  isDocumentsLoading?: boolean;
  isDocumentsFetching?: boolean;
  onSelectSession: (sessionId: number) => void;
  onNewChat: () => void;
  onUploadDocument: (file: File, title: string) => Promise<void>;
  onDeleteDocument: (documentId: number) => Promise<void>;
  onRefreshDocuments: () => void;
  isUploading?: boolean;
}

export function KnowledgeSidebar({
  sessions,
  documents,
  currentSessionId,
  isSessionsLoading,
  isDocumentsLoading,
  isDocumentsFetching,
  onSelectSession,
  onNewChat,
  onUploadDocument,
  onDeleteDocument,
  onRefreshDocuments,
  isUploading,
}: KnowledgeSidebarProps) {
  const [activeTab, setActiveTab] = useState<"chats" | "sources">("chats");

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Tabs */}
      <div className="p-3 border-b border-gray-200">
        <div className="flex rounded-lg p-1" style={{ backgroundColor: "#e5e7eb" }}>
          <button
            onClick={() => setActiveTab("chats")}
            className="flex-1 flex items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            style={{
              backgroundColor: activeTab === "chats" ? "white" : "transparent",
              color: activeTab === "chats" ? "#111827" : "#6b7280",
              boxShadow: activeTab === "chats" ? "0 1px 2px rgba(0,0,0,0.05)" : "none",
            }}
          >
            <MessageSquare className="h-3.5 w-3.5" />
            Chats
          </button>
          <button
            onClick={() => setActiveTab("sources")}
            className="flex-1 flex items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            style={{
              backgroundColor: activeTab === "sources" ? "white" : "transparent",
              color: activeTab === "sources" ? "#111827" : "#6b7280",
              boxShadow: activeTab === "sources" ? "0 1px 2px rgba(0,0,0,0.05)" : "none",
            }}
          >
            <FileText className="h-3.5 w-3.5" />
            Sources
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {activeTab === "chats" ? (
          <ChatsTab
            sessions={sessions}
            currentSessionId={currentSessionId}
            isLoading={isSessionsLoading}
            onSelectSession={onSelectSession}
            onNewChat={onNewChat}
          />
        ) : (
          <SourcesTab
            documents={documents}
            isLoading={isDocumentsLoading}
            isFetching={isDocumentsFetching}
            onUpload={onUploadDocument}
            onDelete={onDeleteDocument}
            onRefresh={onRefreshDocuments}
            isUploading={isUploading}
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
        <Button
          onClick={onNewChat}
          variant="outline"
          size="sm"
          className="w-full gap-2"
        >
          <Plus className="h-4 w-4" />
          New Chat
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
            <p className="mt-2 text-sm text-gray-500">No chats yet</p>
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

function SourcesTab({
  documents,
  isLoading,
  isFetching,
  onUpload,
  onDelete,
  onRefresh,
  isUploading,
}: {
  documents: Document[];
  isLoading?: boolean;
  isFetching?: boolean;
  onUpload: (file: File, title: string) => Promise<void>;
  onDelete: (documentId: number) => Promise<void>;
  onRefresh: () => void;
  isUploading?: boolean;
}) {
  return (
    <>
      <div className="p-3">
        <Dialog>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm" className="w-full gap-2">
              <Upload className="h-4 w-4" />
              Upload
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Upload Document</DialogTitle>
            </DialogHeader>
            <DocumentUpload onUpload={onUpload} isUploading={isUploading} />
          </DialogContent>
        </Dialog>
      </div>

      <div className="flex-1 overflow-y-auto px-3 pb-3">
        <DocumentList
          documents={documents}
          isLoading={isLoading}
          isFetching={isFetching}
          onDelete={onDelete}
          onRefresh={onRefresh}
          compact={true}
        />
      </div>
    </>
  );
}
