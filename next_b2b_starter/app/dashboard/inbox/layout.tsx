import type { Metadata } from "next";
import { PRODUCT_NAME } from "@/lib/brand";

export const metadata: Metadata = {
  title: `Inbox | ${PRODUCT_NAME}`,
};

export default function InboxLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
