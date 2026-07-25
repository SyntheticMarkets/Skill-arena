"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { KeyRound, Lock, LogOut, NotebookPen, RefreshCw, ShieldAlert, Unlock, X } from "lucide-react";
import { api, APIError, dateTime, money } from "@/lib/api";
import type { User, UserRecord } from "@/lib/types";
import { useAuth } from "@/app/auth-provider";
import { Badge, Button, EmptyState, ErrorState, Input, LoadingState, Modal, PageHeader, Panel, SearchInput, Select, Textarea } from "@/components/ui";

type UserList = { users: User[]; total: number };
type UserAction = "status" | "logout" | "note" | "restriction" | "liftRestriction" | "role" | "mfa";

export default function UsersPage() {
  const { can } = useAuth();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<UserRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState("");
  const [action, setAction] = useState<UserAction | null>(null);
  const [restrictionID, setRestrictionID] = useState("");

  const loadUsers = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await api<UserList>(`/api/v1/admin-crm/users?query=${encodeURIComponent(query)}&status=${encodeURIComponent(status)}&limit=100`);
      setUsers(result.users);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "Player records are unavailable.");
    } finally {
      setLoading(false);
    }
  }, [query, status]);

  useEffect(() => {
    const timer = window.setTimeout(() => void loadUsers(), 250);
    return () => window.clearTimeout(timer);
  }, [loadUsers]);

  async function openUser(userID: string) {
    setDetailLoading(true);
    try {
      setSelected(await api<UserRecord>(`/api/v1/admin-crm/users/${userID}`));
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "Player detail is unavailable.");
    } finally {
      setDetailLoading(false);
    }
  }

  async function refreshSelected() {
    if (selected) await openUser(selected.user.id);
    await loadUsers();
  }

  return (
    <>
      <PageHeader eyebrow="Player operations" title="Player records" description="Search identity, security, progression, financial, and compliance state without direct wallet editing." />
      <Panel title={`${total} player records`} description="Results come from the authoritative identity repository." actions={<div className="toolbar"><SearchInput placeholder="Email, name, or ID" value={query} onChange={(event) => setQuery(event.target.value)} /><select aria-label="Account status" value={status} onChange={(event) => setStatus(event.target.value)}><option value="">All statuses</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="disabled">Disabled</option></select></div>}>
        {loading ? <LoadingState label="Searching player records" /> : error ? <ErrorState message={error} retry={() => void loadUsers()} /> : users.length === 0 ? <EmptyState title="No matching players" description="Change the search or status filter." /> : (
          <div className="table-wrap"><table><thead><tr><th>Player</th><th>Country</th><th>Verification</th><th>Role</th><th>Status</th><th>Joined</th></tr></thead><tbody>
            {users.map((user) => <tr key={user.id} tabIndex={0} onClick={() => void openUser(user.id)} onKeyDown={(event) => event.key === "Enter" && void openUser(user.id)}>
              <td><strong>{user.displayName || user.username || "Profile pending"}</strong><span>{user.email}</span></td><td>{user.country || "Not set"}</td>
              <td><Badge tone={user.emailVerified ? "healthy" : "warning"}>{user.emailVerified ? "Email verified" : "Pending email"}</Badge></td>
              <td>{user.role.replaceAll("_", " ")}</td><td><Badge>{user.status}</Badge></td><td>{dateTime(user.createdAt)}</td>
            </tr>)}
          </tbody></table></div>
        )}
      </Panel>
      {detailLoading ? <div className="drawer-backdrop"><aside className="drawer"><LoadingState label="Loading player record" /></aside></div> : null}
      {selected && !detailLoading ? (
        <div className="drawer-backdrop" onMouseDown={(event) => event.target === event.currentTarget && setSelected(null)}>
          <aside className="drawer" aria-label="Player record">
            <div className="drawer-header"><div><p className="eyebrow">Player record</p><h2>{selected.user.displayName || selected.user.email}</h2><p>{selected.user.id}</p></div><button aria-label="Close player record" title="Close player record" onClick={() => setSelected(null)}><X size={18} /></button></div>
            <div className="drawer-actions">
              {can("users.manage") ? <><Button className="button-secondary" onClick={() => setAction("status")}>{selected.user.status === "active" ? <Lock size={15} /> : <Unlock size={15} />}Status</Button><Button className="button-secondary" onClick={() => setAction("logout")}><LogOut size={15} />Force logout</Button><Button className="button-secondary" onClick={() => setAction("note")}><NotebookPen size={15} />Note</Button></> : null}
              {can("compliance.manage") ? <Button className="button-secondary" onClick={() => setAction("restriction")}><ShieldAlert size={15} />Restrict</Button> : null}
              {can("admin_roles.manage") ? <><Button className="button-secondary" onClick={() => setAction("role")}><KeyRound size={15} />Role</Button><Button className="button-secondary" onClick={() => setAction("mfa")}><RefreshCw size={15} />Reset MFA</Button></> : null}
            </div>
            <div className="record-grid">
              <RecordSection title="Identity"><dl className="definition-list"><div><dt>Email</dt><dd>{selected.user.email}</dd></div><div><dt>Country</dt><dd>{selected.user.country || "Not set"}</dd></div><div><dt>KYC</dt><dd>{selected.user.kycStatus}</dd></div><div><dt>Account</dt><dd>{selected.user.status}</dd></div></dl></RecordSection>
              <RecordSection title="Financial"><dl className="definition-list"><div><dt>Available</dt><dd>{money(selected.wallet?.availableMinor || 0, selected.wallet?.currency || "ZAR")}</dd></div><div><dt>Pending deposits</dt><dd>{money(selected.wallet?.pendingDepositMinor || 0, selected.wallet?.currency || "ZAR")}</dd></div><div><dt>Pending withdrawals</dt><dd>{money(selected.wallet?.pendingWithdrawalMinor || 0, selected.wallet?.currency || "ZAR")}</dd></div><div><dt>Risk</dt><dd>{selected.assessment?.riskClassification || "Unassessed"}</dd></div></dl></RecordSection>
              <RecordSection title="Progression"><dl className="definition-list"><div><dt>League</dt><dd>{selected.progression?.leagueTier || "Unranked"}</dd></div><div><dt>Level</dt><dd>{selected.progression?.level || 1}</dd></div><div><dt>Trust</dt><dd>{selected.progression?.trustScore ?? "Not calibrated"}</dd></div><div><dt>Matches</dt><dd>{selected.progression?.matchesPlayed || 0}</dd></div></dl></RecordSection>
              <RecordSection title="Security"><dl className="definition-list"><div><dt>Active sessions</dt><dd>{selected.sessions.filter((item) => !item.revokedAt).length}</dd></div><div><dt>Devices</dt><dd>{selected.devices.length}</dd></div><div><dt>Role</dt><dd>{selected.user.role.replaceAll("_", " ")}</dd></div></dl></RecordSection>
            </div>
            <RecordSection title="Active restrictions">{selected.activeRestrictions.length ? <div className="stack-list">{selected.activeRestrictions.map((item) => <article key={item.id}><div><strong>{item.type}</strong><Badge>{item.status}</Badge></div><p>{item.reason}</p><small>{item.expiresAt ? `Expires ${dateTime(item.expiresAt)}` : "No automatic expiry"}</small>{can("compliance.manage") ? <Button className="button-secondary" onClick={() => { setRestrictionID(item.id); setAction("liftRestriction"); }}>Lift restriction</Button> : null}</article>)}</div> : <p className="muted">No active restrictions.</p>}</RecordSection>
            <RecordSection title="Match history">{selected.matchHistory.length ? <div className="timeline">{selected.matchHistory.slice(0, 20).map((match) => <article key={match.id}><p><strong>{match.gameType}</strong> / {match.mode || "standard"} / {match.outcome || match.state || "recorded"}</p><small>{dateTime(match.completedAt || match.createdAt)}</small></article>)}</div> : <p className="muted">No match history.</p>}</RecordSection>
            <RecordSection title="Financial history"><dl className="definition-list"><div><dt>Deposits</dt><dd>{selected.deposits.length}</dd></div><div><dt>Withdrawals</dt><dd>{selected.withdrawals.length}</dd></div><div><dt>Statement credits</dt><dd>{money(selected.statement?.totalCreditMinor || 0, selected.statement?.currency || "ZAR")}</dd></div><div><dt>Statement debits</dt><dd>{money(selected.statement?.totalDebitMinor || 0, selected.statement?.currency || "ZAR")}</dd></div></dl></RecordSection>
            <RecordSection title="Internal notes">{selected.internalNotes.length ? <div className="timeline">{selected.internalNotes.map((note) => <article key={note.id}><p>{note.body}</p><small>{dateTime(note.createdAt)} / {note.authorId}</small></article>)}</div> : <p className="muted">No internal notes.</p>}</RecordSection>
          </aside>
        </div>
      ) : null}
      {selected && action ? <UserActionModal action={action} restrictionID={restrictionID} record={selected} close={() => setAction(null)} completed={async () => { setAction(null); setRestrictionID(""); await refreshSelected(); }} /> : null}
    </>
  );
}

function RecordSection({ title, children }: { title: string; children: React.ReactNode }) {
  return <section className="record-section"><h3>{title}</h3>{children}</section>;
}

function UserActionModal({ action, restrictionID, record, close, completed }: { action: UserAction; restrictionID: string; record: UserRecord; close: () => void; completed: () => Promise<void> }) {
  const [reason, setReason] = useState("");
  const [body, setBody] = useState("");
  const [status, setStatus] = useState(record.user.status === "active" ? "suspended" : "active");
  const [role, setRole] = useState(record.user.role);
  const [type, setType] = useState("account");
  const [expiresAt, setExpiresAt] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const base = `/api/v1/admin-crm/users/${record.user.id}`;
      if (action === "note") await api(`${base}/notes`, { method: "POST", body: JSON.stringify({ body }) });
      else if (action === "restriction") await api(`${base}/restrictions`, { method: "POST", body: JSON.stringify({ type, reason, expiresAt: expiresAt ? new Date(expiresAt).toISOString() : undefined }) });
      else if (action === "liftRestriction") await api(`${base}/restrictions`, { method: "PATCH", body: JSON.stringify({ restrictionId: restrictionID, reason }) });
      else if (action === "status") await api(`${base}/status`, { method: "POST", body: JSON.stringify({ status, reason }) });
      else if (action === "logout") await api(`${base}/force-logout`, { method: "POST", body: JSON.stringify({ reason }) });
      else if (action === "role") await api(`${base}/role`, { method: "POST", body: JSON.stringify({ role, reason }) });
      else if (action === "mfa") await api(`${base}/mfa/reset`, { method: "POST", body: JSON.stringify({ reason }) });
      await completed();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The operation could not be completed.");
    } finally {
      setSubmitting(false);
    }
  }

  const titles = { status: "Change account status", logout: "Revoke every session", note: "Add internal note", restriction: "Apply restriction", liftRestriction: "Lift restriction", role: "Change administrator role", mfa: "Reset administrator MFA" };
  return <Modal title={titles[action]} description={`Target: ${record.user.email}`} onClose={close}><form className="stack" onSubmit={submit}>
    {action === "status" ? <Select label="New status" value={status} onChange={(event) => setStatus(event.target.value)}><option value="active">Active</option><option value="suspended">Suspended</option><option value="disabled">Disabled</option></Select> : null}
    {action === "role" ? <Select label="Role" value={role} onChange={(event) => setRole(event.target.value)}><option value="player">Player</option><option value="admin">Administrator</option><option value="super_admin">Super administrator</option><option value="treasury_manager">Treasury manager</option><option value="fraud_analyst">Fraud analyst</option><option value="compliance">Compliance</option><option value="finance">Finance</option><option value="support">Support</option><option value="operations">Operations</option><option value="read_only">Read only</option></Select> : null}
    {action === "restriction" ? <><Select label="Restriction" value={type} onChange={(event) => setType(event.target.value)}><option value="account">Account</option><option value="deposit">Deposits</option><option value="withdrawal">Withdrawals</option><option value="competition">Competition</option><option value="communication">Communication</option><option value="cooling_off">Cooling-off</option><option value="self_exclusion">Self-exclusion</option></Select><Input label={type === "cooling_off" || type === "self_exclusion" ? "Expiry" : "Expiry (optional)"} type="datetime-local" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} required={type === "cooling_off" || type === "self_exclusion"} /></> : null}
    {action === "note" ? <Textarea label="Internal note" value={body} onChange={(event) => setBody(event.target.value)} minLength={1} maxLength={4000} required /> : <Textarea label="Mandatory reason" value={reason} onChange={(event) => setReason(event.target.value)} minLength={4} maxLength={1000} required />}
    {error ? <p className="form-error" role="alert">{error}</p> : null}<div className="button-row"><Button type="button" className="button-secondary" onClick={close}>Cancel</Button><Button type="submit" disabled={submitting}>{submitting ? "Applying..." : "Confirm operation"}</Button></div>
  </form></Modal>;
}
