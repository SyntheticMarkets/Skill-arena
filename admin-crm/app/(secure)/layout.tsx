import { AdminShell } from "@/components/admin-shell";

export default function SecureLayout({ children }: { children: React.ReactNode }) {
  return <AdminShell>{children}</AdminShell>;
}
