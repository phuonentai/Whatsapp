import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Inbox | AP Cash",
};

export default function InboxLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
