"use client";

import { Activity, AlertTriangle, Database, HardDrive, Mail, RefreshCw, Server, WalletCards, Workflow } from "lucide-react";
import type { Monitoring } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { Badge, Button, ErrorState, LoadingState, Metric, PageHeader, Panel } from "@/components/ui";

export default function MonitoringPage() {
  const resource = useResource<Monitoring>("/api/v1/admin-crm/monitoring");
  if (resource.loading) return <LoadingState />;
  if (resource.error || !resource.data) return <ErrorState message={resource.error} retry={() => void resource.reload()} />;
  const system = resource.data.system;
  const services = [
    ["API", system.apiStatus, Server],
    ["PostgreSQL", resource.data.dependencies.database, Database],
    ["Redis cache", resource.data.dependencies.redis, Activity],
    ["Object storage", resource.data.dependencies.storage, HardDrive],
    ["Payment providers", resource.data.dependencies.paymentProviders, WalletCards],
    ["Email service", resource.data.dependencies.email, Mail],
    ["Queue", system.queueStatus, Workflow],
    ["Backups", system.backupStatus, HardDrive]
  ] as const;
  return (
    <>
      <PageHeader eyebrow="Read-only monitoring" title="System health" description="Dependency, queue, worker, storage, and deployment state. Operational restart controls are intentionally unavailable." actions={<Button className="button-secondary" onClick={() => void resource.reload()}><RefreshCw size={16} />Refresh</Button>} />
      <div className="health-grid">{services.map(([label, status, Icon]) => <div className="health-service" key={label}><Icon size={20} /><div><span>{label}</span><strong>{String(status || "unknown")}</strong></div><Badge tone={status === "ok" || status === "healthy" ? "healthy" : "warning"}>{String(status || "unknown")}</Badge></div>)}</div>
      {resource.data.alerts.length ? <Panel title="Active alerts" description="Dependency checks requiring operator attention"><div className="stack-list">{resource.data.alerts.map((alert) => <article key={alert}><div><strong><AlertTriangle size={16} />{alert}</strong></div></article>)}</div></Panel> : null}
      <div className="two-column">
        <Panel title="Queue position" description="Background coordination and recovery workload"><div className="mini-metrics"><Metric label="Pending" value={resource.data.queue.pendingJobs} /><Metric label="Running" value={resource.data.queue.runningJobs} /><Metric label="Failed" value={resource.data.queue.failedJobs} tone={resource.data.queue.failedJobs ? "danger" : "default"} /><Metric label="Retries" value={resource.data.queue.retryCount} /></div></Panel>
        <Panel title="Worker state" description="Latest heartbeat reported by each worker"><div className="health-list">{Object.entries(resource.data.queue.workerStatus || {}).map(([name, status]) => <div key={name}><span>{name.replaceAll("_", " ")}</span><Badge tone={status === "healthy" || status === "ok" ? "healthy" : "warning"}>{status}</Badge></div>)}</div></Panel>
      </div>
    </>
  );
}
