"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Copy, KeyRound, ShieldCheck } from "lucide-react";
import { api, APIError } from "@/lib/api";
import { Button, IconButton, Input } from "@/components/ui";
import { useAuth } from "@/app/auth-provider";

export default function AdminMFASetupPage() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [secret, setSecret] = useState("");
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ secret: string }>("/api/v1/admin-crm/auth/mfa/setup", { method: "POST" })
      .then((result) => setSecret(result.secret))
      .catch((requestError) => setError(requestError instanceof APIError ? requestError.message : "MFA enrollment is unavailable."));
  }, []);

  async function confirm(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      const result = await api<{ recoveryCodes: string[] }>("/api/v1/admin-crm/auth/mfa/confirm", {
        method: "POST",
        body: JSON.stringify({ code })
      });
      setRecoveryCodes(result.recoveryCodes);
      await refresh();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The code could not be verified.");
    }
  }

  async function copyCodes() {
    await navigator.clipboard.writeText(recoveryCodes.join("\n"));
  }

  return (
    <main className="auth-page auth-page-compact">
      <section className="auth-context">
        <div className="auth-brand"><span><ShieldCheck size={22} /></span><strong>Skill Arena</strong></div>
        <div><p className="eyebrow">Required enrollment</p><h1>Protect the operations account</h1><p>Administrator access remains restricted until a time-based authenticator is enrolled.</p></div>
      </section>
      <section className="auth-form-wrap">
        <div className="auth-form">
          <div className="auth-icon"><KeyRound /></div>
          {recoveryCodes.length ? (
            <>
              <div><p className="eyebrow">Enrollment complete</p><h2>Store recovery codes</h2><p>Each code works once. Keep them outside this device and never share them.</p></div>
              <div className="recovery-grid">{recoveryCodes.map((item) => <code key={item}>{item}</code>)}</div>
              <div className="button-row"><Button onClick={() => void copyCodes()}><Copy size={16} />Copy codes</Button><Button className="button-secondary" onClick={() => router.replace("/")}>Enter operations</Button></div>
            </>
          ) : (
            <form onSubmit={confirm} className="stack">
              <div><p className="eyebrow">Authenticator setup</p><h2>Add the secret</h2><p>Enter this key in your authenticator, then confirm the generated six-digit code.</p></div>
              <div className="secret-row"><code>{secret || "Generating secure secret..."}</code><IconButton label="Copy secret" disabled={!secret} onClick={() => void navigator.clipboard.writeText(secret)}><Copy size={16} /></IconButton></div>
              <Input label="Six-digit code" inputMode="numeric" autoComplete="one-time-code" maxLength={6} value={code} onChange={(event) => setCode(event.target.value)} required />
              {error ? <p className="form-error" role="alert">{error}</p> : null}
              <Button type="submit" disabled={!secret}>Confirm authenticator</Button>
            </form>
          )}
        </div>
      </section>
    </main>
  );
}
