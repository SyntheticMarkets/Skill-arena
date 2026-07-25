"use client";

import { FormEvent, useState } from "react";
import { FileCheck2, Scale, ShieldAlert } from "lucide-react";
import { api, apiURL, APIError, money } from "@/lib/api";
import type { ComplianceCase, Jurisdiction } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { useAuth } from "@/app/auth-provider";
import { Badge, Button, EmptyState, ErrorState, Input, LoadingState, Modal, PageHeader, Panel, Select, Textarea } from "@/components/ui";

export default function CompliancePage() {
  const { can } = useAuth();
  const cases = useResource<{ cases: ComplianceCase[] }>("/api/v1/admin-crm/compliance/cases");
  const jurisdictions = useResource<{ jurisdictions: Jurisdiction[] }>("/api/v1/admin-crm/compliance/jurisdictions");
  const [selectedCase, setSelectedCase] = useState<ComplianceCase | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<Jurisdiction | null>(null);

  return (
    <>
      <PageHeader eyebrow="Compliance center" title="Identity, risk, and jurisdiction" description="Review evidence and control launch policy without exposing internal decisions to players." />
      <div className="two-column compliance-layout">
        <Panel title="KYC and AML queue" description="Submitted assessments, evidence, provider responses, and review signals">
          {cases.loading ? <LoadingState /> : cases.error || !cases.data ? <ErrorState message={cases.error} retry={() => void cases.reload()} /> : cases.data.cases.length === 0 ? <EmptyState title="Review queue clear" description="No identity or financial assessments require review." /> : <div className="case-list">{cases.data.cases.map((item) => <button key={item.user.id} onClick={() => setSelectedCase(item)}><div><strong>{item.user.displayName || item.user.email}</strong><span>{item.user.email}</span></div><div><Badge>{item.assessment?.status || "not started"}</Badge><span>{item.evidence.length} evidence</span></div></button>)}</div>}
        </Panel>
        <Panel title="Jurisdiction policy" description="Effective operational controls stored in PostgreSQL">
          {jurisdictions.loading ? <LoadingState /> : jurisdictions.error || !jurisdictions.data ? <ErrorState message={jurisdictions.error} retry={() => void jurisdictions.reload()} /> : jurisdictions.data.jurisdictions.length === 0 ? <EmptyState title="No runtime policies recorded" description="Deployment policy remains authoritative until an approved jurisdiction policy is saved." /> : <div className="stack-list">{jurisdictions.data.jurisdictions.map((item) => <article key={item.country}><div><strong>{item.country} · {item.currency}</strong><Badge tone={item.depositEnabled && item.withdrawalEnabled ? "healthy" : "warning"}>{item.depositEnabled && item.withdrawalEnabled ? "Live enabled" : "Restricted"}</Badge></div><dl className="inline-definitions"><div><dt>Daily deposit</dt><dd>{money(item.dailyDepositMinor, item.currency)}</dd></div><div><dt>Daily withdrawal</dt><dd>{money(item.dailyWithdrawalMinor, item.currency)}</dd></div><div><dt>Minimum age</dt><dd>{item.minimumAge}</dd></div></dl>{can("compliance.manage") ? <Button className="button-secondary" onClick={() => setSelectedPolicy(item)}>Edit policy</Button> : null}</article>)}</div>}
          {can("compliance.manage") ? <Button className="button-secondary full-width" onClick={() => setSelectedPolicy(emptyPolicy())}>Add jurisdiction policy</Button> : null}
        </Panel>
      </div>
      {selectedCase ? <CaseModal item={selectedCase} close={() => setSelectedCase(null)} complete={async () => { setSelectedCase(null); await cases.reload(); }} canDecide={can("kyc.decide")} /> : null}
      {selectedPolicy ? <PolicyModal item={selectedPolicy} close={() => setSelectedPolicy(null)} complete={async () => { setSelectedPolicy(null); await jurisdictions.reload(); }} /> : null}
    </>
  );
}

function emptyPolicy(): Jurisdiction {
  return { country: "", currency: "ZAR", minimumAge: 18, depositEnabled: false, withdrawalEnabled: false, sourceOfFundsRequired: true, dailyDepositMinor: 0, monthlyDepositMinor: 0, dailyWithdrawalMinor: 0, monthlyWithdrawalMinor: 0 };
}

function CaseModal({ item, close, complete, canDecide }: { item: ComplianceCase; close: () => void; complete: () => Promise<void>; canDecide: boolean }) {
  const [decision, setDecision] = useState("in_review");
  const [risk, setRisk] = useState(item.assessment?.riskClassification || "standard");
  const [verification, setVerification] = useState(item.assessment?.verificationStatus || "pending");
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      await api("/api/v1/admin-crm/compliance/decisions", { method: "POST", body: JSON.stringify({ userId: item.user.id, decision, riskClassification: risk, verificationStatus: verification, reason }) });
      await complete();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The compliance decision could not be recorded.");
    }
  }
  return <Modal title="Compliance review" description={item.user.email} onClose={close}><div className="stack">
    <div className="record-grid"><section className="record-section"><h3>Assessment</h3><dl className="definition-list"><div><dt>Status</dt><dd>{item.assessment?.status || "Not started"}</dd></div><div><dt>Country</dt><dd>{item.assessment?.country || item.user.country}</dd></div><div><dt>Occupation</dt><dd>{item.assessment?.occupation || "Not supplied"}</dd></div><div><dt>Source of funds</dt><dd>{item.assessment?.sourceOfFunds || "Not supplied"}</dd></div></dl></section><section className="record-section"><h3>Evidence</h3>{item.evidence.length ? item.evidence.map((evidence) => <a className="evidence-row" key={evidence.id} href={apiURL(`/api/v1/admin-crm/compliance/evidence/${evidence.id}`)} target="_blank" rel="noreferrer"><FileCheck2 size={16} /><div><strong>{evidence.type}</strong><span>{evidence.contentType} / {Math.ceil(evidence.sizeBytes / 1024)} KB</span></div><Badge>{evidence.status}</Badge></a>) : <p className="muted">No evidence uploaded.</p>}</section><section className="record-section"><h3>Provider responses</h3>{item.providerResponses.length ? <div className="stack-list">{item.providerResponses.map((response) => <article key={response.id}><div><strong>{response.provider} / {response.checkType}</strong><Badge tone={response.status === "clear" ? "healthy" : response.status === "review" ? "warning" : undefined}>{response.status}</Badge></div><p>Reference {response.providerReference}</p><small>{response.riskSignals.length ? response.riskSignals.join(", ") : "No risk signals returned"}</small></article>)}</div> : <p className="muted">No provider responses received.</p>}</section></div>
    {canDecide ? <form className="stack" onSubmit={submit}><Select label="Decision" value={decision} onChange={(event) => setDecision(event.target.value)}><option value="in_review">Request more information</option><option value="complete">Approve assessment</option><option value="restricted">Restrict account</option></Select><Select label="Risk classification" value={risk} onChange={(event) => setRisk(event.target.value)}><option value="low">Low</option><option value="standard">Standard</option><option value="elevated">Elevated</option><option value="high">High</option></Select><Select label="Verification status" value={verification} onChange={(event) => setVerification(event.target.value)}><option value="pending">Pending</option><option value="verified">Verified</option><option value="rejected">Rejected</option><option value="more_information">More information required</option></Select><Textarea label="Mandatory decision reason" value={reason} onChange={(event) => setReason(event.target.value)} required minLength={4} maxLength={1000} />{error ? <p className="form-error">{error}</p> : null}<div className="button-row"><Button type="button" className="button-secondary" onClick={close}>Cancel</Button><Button type="submit"><Scale size={16} />Record decision</Button></div></form> : <div className="decision-warning"><ShieldAlert /><p>Your role can inspect this case but cannot record compliance decisions.</p></div>}
  </div></Modal>;
}

function PolicyModal({ item, close, complete }: { item: Jurisdiction; close: () => void; complete: () => Promise<void> }) {
  const [policy, setPolicy] = useState(item);
  const [error, setError] = useState("");
  function number(name: keyof Jurisdiction, value: string) {
    setPolicy((current) => ({ ...current, [name]: Number(value) }));
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      await api("/api/v1/admin-crm/compliance/jurisdictions", { method: "PUT", body: JSON.stringify(policy) });
      await complete();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The policy could not be saved.");
    }
  }
  return <Modal title="Jurisdiction policy" description="Changes apply to the backend policy boundary after approval." onClose={close}><form className="stack" onSubmit={submit}><div className="form-grid"><Input label="Country code" value={policy.country} onChange={(event) => setPolicy({ ...policy, country: event.target.value.toUpperCase() })} minLength={2} maxLength={2} required /><Input label="Currency" value={policy.currency} onChange={(event) => setPolicy({ ...policy, currency: event.target.value.toUpperCase() })} minLength={3} maxLength={3} required /><Input label="Minimum age" type="number" min={18} value={policy.minimumAge} onChange={(event) => number("minimumAge", event.target.value)} required /><Input label="Daily deposit (minor)" type="number" min={0} value={policy.dailyDepositMinor} onChange={(event) => number("dailyDepositMinor", event.target.value)} required /><Input label="Monthly deposit (minor)" type="number" min={0} value={policy.monthlyDepositMinor} onChange={(event) => number("monthlyDepositMinor", event.target.value)} required /><Input label="Daily withdrawal (minor)" type="number" min={0} value={policy.dailyWithdrawalMinor} onChange={(event) => number("dailyWithdrawalMinor", event.target.value)} required /><Input label="Monthly withdrawal (minor)" type="number" min={0} value={policy.monthlyWithdrawalMinor} onChange={(event) => number("monthlyWithdrawalMinor", event.target.value)} required /></div><div className="check-grid"><label><input type="checkbox" checked={policy.depositEnabled} onChange={(event) => setPolicy({ ...policy, depositEnabled: event.target.checked })} />Deposits enabled</label><label><input type="checkbox" checked={policy.withdrawalEnabled} onChange={(event) => setPolicy({ ...policy, withdrawalEnabled: event.target.checked })} />Withdrawals enabled</label><label><input type="checkbox" checked={policy.sourceOfFundsRequired} onChange={(event) => setPolicy({ ...policy, sourceOfFundsRequired: event.target.checked })} />Source of funds required</label></div>{error ? <p className="form-error">{error}</p> : null}<div className="button-row"><Button type="button" className="button-secondary" onClick={close}>Cancel</Button><Button type="submit">Save approved policy</Button></div></form></Modal>;
}
