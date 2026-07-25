import type { Metadata } from "next";
import { AuthProvider } from "./auth-provider";
import "./styles.css";

export const metadata: Metadata = {
  title: "Skill Arena Operations",
  description: "Authorized operations and compliance workspace"
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
