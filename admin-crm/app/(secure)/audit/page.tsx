"use client";

import { Fingerprint, Link2, ShieldCheck, ShieldX } from "lucide-react";
import { dateTime } from "@/lib/api";
import type { AuditLog } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { Badge, EmptyState, ErrorState, LoadingState, PageHeader, Panel } from "@/components/ui";

export default function AuditPage() {
  const resource = useResource<{ logs: AuditLog[]; chainStatus: string }>("/api/v1/admin-crm/audit?limit=200");
  return (
    <>
      <PageHeader eyebrow="Immutable evidence" title="Audit center" description="Administrator attribution, affected resources, reasons, network context, and hash-chain identity." />
      {resource.loading ? <LoadingState /> : resource.error || !resource.data ? <ErrorState message={resource.error} retry={() => void resource.reload()} /> : (
        <>
          <div className={`audit-integrity ${resource.data.chainStatus === "verified" ? "" : "invalid"}`}>{resource.data.chainStatus === "verified" ? <ShieldCheck /> : <ShieldX />}<div><strong>Audit chain {resource.data.chainStatus}</strong><p>{resource.data.chainStatus === "verified" ? "Every returned record has an immutable entry hash." : "Audit integrity requires immediate security investigation."}</p></div></div>
          <Panel title={`${resource.data.logs.length} recent events`} description="Audit records cannot be edited or deleted from the CRM.">
            {resource.data.logs.length === 0 ? <EmptyState title="No audit events" description="No events are available in the requested window." /> : <div className="audit-list">{resource.data.logs.map((log) => <article key={log.id}><div className="audit-icon"><Fingerprint size={17} /></div><div><div><strong>{log.action.replaceAll(".", " ")}</strong><Badge>{log.metadata?.reason ? "Reason recorded" : "System event"}</Badge></div><p>Actor <code>{log.actorId}</code>{log.targetId ? <> affected <code>{log.targetId}</code></> : null}</p>{log.metadata?.reason ? <blockquote>{log.metadata.reason}</blockquote> : null}<div className="audit-meta"><span>{dateTime(log.createdAt)}</span><span>{log.ipAddress || "IP unavailable"}</span>{log.entryHash ? <span><Link2 size={12} />{log.entryHash.slice(0, 16)}</span> : null}</div></div></article>)}</div>}
          </Panel>
        </>
      )}
    </>
  );
}
