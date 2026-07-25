"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, LockKeyhole, ShieldCheck } from "lucide-react";
import { api, APIError } from "@/lib/api";
import { Button, Input } from "@/components/ui";
import { useAuth } from "../auth-provider";

type LoginResult = {
  authenticated?: boolean;
  mfaRequired?: boolean;
  challengeToken?: string;
  mfaEnrollmentRequired?: boolean;
};

export default function AdminLoginPage() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const result = await api<LoginResult>("/api/v1/admin-crm/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password })
      });
      if (result.mfaRequired && result.challengeToken) {
        sessionStorage.setItem("admin_mfa_challenge", result.challengeToken);
        router.push("/mfa");
        return;
      }
      await refresh();
      router.push(result.mfaEnrollmentRequired ? "/mfa/setup" : "/");
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The administrator sign-in service is unavailable.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-context" aria-label="Operations security">
        <div className="auth-brand"><span><ShieldCheck size={22} /></span><strong>Skill Arena</strong></div>
        <div>
          <p className="eyebrow">Restricted workspace</p>
          <h1>Operations command center</h1>
          <p>Financial, compliance, support, and platform controls for authorized personnel. Every action is permission-checked and immutably audited.</p>
        </div>
        <ul>
          <li><LockKeyhole size={17} />Dedicated administrator sessions</li>
          <li><KeyRound size={17} />Mandatory multi-factor authentication</li>
          <li><ShieldCheck size={17} />Least-privilege operational access</li>
        </ul>
      </section>
      <section className="auth-form-wrap">
        <form className="auth-form" onSubmit={submit}>
          <div>
            <p className="eyebrow">Administrator access</p>
            <h2>Sign in</h2>
            <p>Use your approved staff identity. Player accounts cannot access this application.</p>
          </div>
          <Input label="Work email" type="email" autoComplete="username" value={email} onChange={(event) => setEmail(event.target.value)} required />
          <Input label="Password" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          {error ? <p className="form-error" role="alert">{error}</p> : null}
          <Button type="submit" disabled={submitting}>{submitting ? "Verifying identity..." : "Continue securely"}</Button>
          <p className="auth-legal">Access is logged. Unauthorized use may result in account suspension and investigation.</p>
        </form>
      </section>
    </main>
  );
}
