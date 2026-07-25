"use client";

import { FormEvent, useState } from "react";
import { Check, RefreshCw, ShieldAlert, X } from "lucide-react";
import { api, APIError, dateTime, money } from "@/lib/api";
import type { FinanceWorkspace, FinancialItem } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { Badge, Button, EmptyState, ErrorState, LoadingState, Modal, PageHeader, Panel, Textarea } from "@/components/ui";

export default function FinancePage() {
  const [filter, setFilter] = useState("");
  const resource = useResource<FinanceWorkspace>(`/api/v1/admin-crm/finance?status=${filter}`);
  const [selected, setSelected] = useState<FinancialItem | null>(null);
  const [decision, setDecision] = useState<"approve" | "reject">("approve");

  return (
    <>
      <PageHeader eyebrow="Financial operations" title="Money movement" description="Review provider-independent deposits, withdrawals, reserve evidence, and reconciliation. Ledger balances cannot be edited here." actions={<Button className="button-secondary" onClick={() => void resource.reload()}><RefreshCw size={16} />Refresh</Button>} />
      <div className="tab-row" role="group" aria-label="Financial status filter">
        {["", "pending_review", "completed", "failed"].map((value) => <button key={value || "all"} className={filter === value ? "active" : ""} onClick={() => setFilter(value)}>{value ? value.replaceAll("_", " ") : "All"}</button>)}
      </div>
      {resource.loading ? <LoadingState /> : resource.error || !resource.data ? <ErrorState message={resource.error} retry={() => void resource.reload()} /> : (
        <div className="stack">
          <Panel title="Withdrawal review" description="Manual approval remains mandatory. Rejections require an internal reason.">
            {resource.data.withdrawals.length === 0 ? <EmptyState title="No withdrawals in this view" description="There is no financial work matching this status." /> : <FinancialTable items={resource.data.withdrawals} review={(item, next) => { setSelected(item); setDecision(next); }} />}
          </Panel>
          <Panel title="Deposit lifecycle" description="Provider callbacks and settlement determine these states.">
            {resource.data.deposits.length === 0 ? <EmptyState title="No deposits in this view" description="No deposit records match the selected status." /> : <FinancialTable items={resource.data.deposits} />}
          </Panel>
          <Panel title="Provider health" description="Live status from the provider-independent Payment Core">
            {resource.data.providers.length ? <div className="health-list">{resource.data.providers.map((provider) => <div key={provider.id}><span>{provider.id}</span><Badge tone={provider.status === "healthy" ? "healthy" : "danger"}>{provider.status}</Badge>{provider.details ? <small>{provider.details}</small> : null}</div>)}</div> : <EmptyState title="No provider configured" description="A production payment adapter must be enabled before launch." />}
          </Panel>
          <div className="two-column">
            <Panel title="Reconciliation" description="Immutable provider-to-journal comparisons">
              {resource.data.reconciliations.length ? <div className="stack-list">{resource.data.reconciliations.map((item) => <article key={item.id}><div><strong>{item.provider} · {item.currency}</strong><Badge>{item.status}</Badge></div><dl className="inline-definitions"><div><dt>Provider</dt><dd>{money(item.providerBalanceMinor, item.currency)}</dd></div><div><dt>Journal</dt><dd>{money(item.journalBalanceMinor, item.currency)}</dd></div><div><dt>Variance</dt><dd>{money(item.differenceMinor, item.currency)}</dd></div></dl><small>{dateTime(item.createdAt)} · {item.immutableHash.slice(0, 12)}</small></article>)}</div> : <EmptyState title="No reconciliation runs" description="No provider reconciliation has been recorded yet." />}
            </Panel>
            <Panel title="Reserve validation" description="Settlement gates recorded by Treasury">
              {resource.data.reserveChecks.length ? <div className="stack-list">{resource.data.reserveChecks.map((item) => <article key={item.id}><div><strong>{item.purpose.replaceAll("_", " ")}</strong><Badge tone={item.passed ? "healthy" : "danger"}>{item.passed ? "Passed" : "Blocked"}</Badge></div><p>{money(item.requestedMinor, item.currency)} requested against {money(item.liabilityMinor, item.currency)} liabilities.</p><small>{item.provider} · {dateTime(item.createdAt)}</small></article>)}</div> : <EmptyState title="No reserve checks" description="Reserve evidence appears when settlement or reconciliation runs." />}
            </Panel>
          </div>
        </div>
      )}
      {selected ? <DecisionModal item={selected} decision={decision} close={() => setSelected(null)} complete={async () => { setSelected(null); await resource.reload(); }} /> : null}
    </>
  );
}

function FinancialTable({ items, review }: { items: FinancialItem[]; review?: (item: FinancialItem, decision: "approve" | "reject") => void }) {
  return <div className="table-wrap"><table><thead><tr><th>Reference</th><th>Player</th><th>Amount</th><th>Method</th><th>Provider</th><th>Status</th><th>Requested</th>{review ? <th>Decision</th> : null}</tr></thead><tbody>
    {items.map((item) => <tr key={item.id}><td><strong>{item.id}</strong></td><td>{item.userId}</td><td>{money(item.amountMinor, item.currency)}</td><td>{item.method.replaceAll("_", " ")}</td><td>{item.provider || "Routing pending"}</td><td><Badge>{item.status}</Badge></td><td>{dateTime(item.requestedAt)}</td>{review ? <td>{item.status === "pending_review" ? <div className="row-actions"><button title="Approve withdrawal" aria-label="Approve withdrawal" onClick={() => review(item, "approve")}><Check size={16} /></button><button title="Reject withdrawal" aria-label="Reject withdrawal" className="danger" onClick={() => review(item, "reject")}><X size={16} /></button></div> : <span className="muted">No action</span>}</td> : null}</tr>)}
  </tbody></table></div>;
}

function DecisionModal({ item, decision, close, complete }: { item: FinancialItem; decision: "approve" | "reject"; close: () => void; complete: () => Promise<void> }) {
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    try {
      await api(`/api/v1/admin-crm/finance/withdrawals/${item.id}/decision`, { method: "POST", body: JSON.stringify({ decision, reason }) });
      await complete();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The withdrawal decision could not be recorded.");
    } finally {
      setSubmitting(false);
    }
  }
  return <Modal title={`${decision === "approve" ? "Approve" : "Reject"} withdrawal`} description={`${money(item.amountMinor, item.currency)} · ${item.id}`} onClose={close}><form className="stack" onSubmit={submit}><div className={`decision-warning ${decision === "reject" ? "danger" : ""}`}>{decision === "approve" ? <Check /> : <ShieldAlert />}<p>{decision === "approve" ? "Approval advances the request to provider processing. It does not settle funds directly." : "The player receives a generic rejection status. This internal reason remains restricted to authorized staff."}</p></div><Textarea label="Mandatory internal reason" value={reason} onChange={(event) => setReason(event.target.value)} minLength={4} maxLength={1000} required />{error ? <p className="form-error" role="alert">{error}</p> : null}<div className="button-row"><Button className="button-secondary" type="button" onClick={close}>Cancel</Button><Button type="submit" className={decision === "reject" ? "button-danger" : ""} disabled={submitting}>{submitting ? "Recording..." : `Confirm ${decision}`}</Button></div></form></Modal>;
}
