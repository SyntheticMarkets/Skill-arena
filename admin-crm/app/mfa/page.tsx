"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound, ShieldCheck } from "lucide-react";
import { api, APIError } from "@/lib/api";
import { Button, Input } from "@/components/ui";
import { useAuth } from "../auth-provider";

export default function AdminMFAPage() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const value = sessionStorage.getItem("admin_mfa_challenge") || "";
      if (!value) router.replace("/login");
      setChallenge(value);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [router]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      await api("/api/v1/admin-crm/auth/mfa/challenge", {
        method: "POST",
        body: JSON.stringify({ challengeToken: challenge, [useRecovery ? "recoveryCode" : "code"]: code })
      });
      sessionStorage.removeItem("admin_mfa_challenge");
      await refresh();
      router.replace("/");
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "MFA verification is unavailable.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="auth-page auth-page-compact">
      <section className="auth-context">
        <div className="auth-brand"><span><ShieldCheck size={22} /></span><strong>Skill Arena</strong></div>
        <div><p className="eyebrow">Second factor</p><h1>Verify privileged access</h1><p>Your password was accepted. Complete the administrator MFA challenge to create a dedicated operations session.</p></div>
      </section>
      <section className="auth-form-wrap">
        <form className="auth-form" onSubmit={submit}>
          <div className="auth-icon"><KeyRound /></div>
          <div><p className="eyebrow">Security challenge</p><h2>{useRecovery ? "Recovery code" : "Authenticator code"}</h2><p>{useRecovery ? "Enter one unused recovery code." : "Enter the six-digit code from your authenticator."}</p></div>
          <Input label={useRecovery ? "Recovery code" : "Six-digit code"} inputMode={useRecovery ? "text" : "numeric"} autoComplete="one-time-code" value={code} onChange={(event) => setCode(event.target.value)} maxLength={useRecovery ? 32 : 6} required />
          {error ? <p className="form-error" role="alert">{error}</p> : null}
          <Button type="submit" disabled={submitting || !challenge}>{submitting ? "Verifying..." : "Verify access"}</Button>
          <button className="text-button" type="button" onClick={() => { setUseRecovery((value) => !value); setCode(""); setError(""); }}>{useRecovery ? "Use authenticator code" : "Use a recovery code"}</button>
        </form>
      </section>
    </main>
  );
}
