import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Knowledge Base",
  description: "Upload documents and chat with AI to find information quickly.",
};

export default function KnowledgeLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
