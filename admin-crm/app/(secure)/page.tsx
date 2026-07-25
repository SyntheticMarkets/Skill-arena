"use client";

import Link from "next/link";
import { ArrowRight, CircleAlert, ShieldCheck } from "lucide-react";
import { dateTime, money } from "@/lib/api";
import type { Dashboard } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { Badge, ErrorState, LoadingState, Metric, PageHeader, Panel } from "@/components/ui";

export default function DashboardPage() {
  const { data, loading, error, reload } = useResource<Dashboard>("/api/v1/admin-crm/dashboard");
  if (loading) return <LoadingState />;
  if (error || !data) return <ErrorState message={error} retry={() => void reload()} />;

  const reviewTotal = data.financial.pendingWithdrawals + data.compliance.pendingKyc + data.support.escalatedTickets;
  return (
    <>
      <PageHeader eyebrow="Live operations" title="Command center" description={`Authoritative platform position as of ${dateTime(data.generatedAt)}.`} />
      <div className="metric-grid metric-grid-primary">
        <Metric label="Players online" value={data.players.onlineUsers} detail={`${data.players.totalUsers} registered`} tone="good" />
        <Metric label="Pending withdrawals" value={data.financial.pendingWithdrawals} detail={`${data.financial.completedWithdrawals} completed`} tone={data.financial.pendingWithdrawals ? "warning" : "default"} />
        <Metric label="Compliance queue" value={data.compliance.pendingKyc + data.compliance.pendingReviews} detail="Identity and risk decisions" tone={data.compliance.pendingKyc ? "warning" : "default"} />
        <Metric label="Escalated support" value={data.support.escalatedTickets} detail={`${data.support.openTickets} open tickets`} tone={data.support.escalatedTickets ? "danger" : "default"} />
      </div>
      <div className="dashboard-grid">
        <Panel title="Priority queue" description="Work requiring an authorized decision">
          {reviewTotal === 0 ? <div className="compact-empty"><ShieldCheck size={20} /><span>No escalated work is waiting.</span></div> : (
            <div className="work-list">
              {data.financial.pendingWithdrawals ? <Link href="/finance"><span><CircleAlert size={17} />Withdrawal review</span><strong>{data.financial.pendingWithdrawals}</strong><ArrowRight size={16} /></Link> : null}
              {data.compliance.pendingKyc ? <Link href="/compliance"><span><CircleAlert size={17} />KYC decisions</span><strong>{data.compliance.pendingKyc}</strong><ArrowRight size={16} /></Link> : null}
              {data.support.escalatedTickets ? <Link href="/support"><span><CircleAlert size={17} />Escalated tickets</span><strong>{data.support.escalatedTickets}</strong><ArrowRight size={16} /></Link> : null}
            </div>
          )}
        </Panel>
        <Panel title="Financial position" description="Provider-independent operational totals">
          <dl className="definition-list">
            <div><dt>Deposits settled today</dt><dd>{money(data.financial.depositsTodayMinor, data.financial.currency)}</dd></div>
            <div><dt>Treasury available</dt><dd>{money(data.financial.treasuryAvailableMinor, data.financial.currency)}</dd></div>
            <div><dt>Active providers</dt><dd>{data.financial.activePaymentProviders}</dd></div>
          </dl>
        </Panel>
        <Panel title="Competition activity" description="Read-only live platform state">
          <div className="mini-metrics">
            <Metric label="Live matches" value={data.games.liveMatches} />
            <Metric label="Active tournaments" value={data.games.activeTournaments} />
            <Metric label="Queue depth" value={data.games.queueSize} />
          </div>
        </Panel>
        <Panel title="Dependency health" description="No restart controls are exposed">
          <div className="health-list">
            {Object.entries(data.system).map(([name, status]) => <div key={name}><span>{name}</span><Badge tone={status === "ok" ? "healthy" : "danger"}>{status}</Badge></div>)}
          </div>
        </Panel>
      </div>
    </>
  );
}
