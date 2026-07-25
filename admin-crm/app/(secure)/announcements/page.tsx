"use client";

import { FormEvent, useState } from "react";
import { BellRing, Send } from "lucide-react";
import { api, APIError, dateTime } from "@/lib/api";
import { useResource } from "@/lib/use-resource";
import { Badge, Button, EmptyState, ErrorState, LoadingState, PageHeader, Panel, Select, Textarea, Input } from "@/components/ui";

type Announcement = { id: string; category: string; title: string; message: string; audience: string; status: string; createdBy: string; createdAt: string; sentAt?: string };

export default function AnnouncementsPage() {
  const resource = useResource<{ announcements: Announcement[] }>("/api/v1/admin-crm/announcements");
  const [category, setCategory] = useState("announcement");
  const [audience, setAudience] = useState("all");
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      await api("/api/v1/admin-crm/announcements", { method: "POST", body: JSON.stringify({ category, audience, title, message }) });
      setTitle("");
      setMessage("");
      await resource.reload();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The notice could not be sent.");
    } finally {
      setSubmitting(false);
    }
  }
  return (
    <>
      <PageHeader eyebrow="Platform communication" title="Notices and announcements" description="Send maintenance, security, compliance, and general messages through the frozen notification service." />
      <div className="two-column announcement-layout">
        <Panel title="Compose notice" description="Messages are delivered to authoritative player notification records.">
          <form className="stack" onSubmit={submit}><div className="form-grid"><Select label="Category" value={category} onChange={(event) => setCategory(event.target.value)}><option value="announcement">Announcement</option><option value="maintenance">Maintenance</option><option value="security">Security</option><option value="compliance">Compliance</option></Select><Select label="Audience" value={audience} onChange={(event) => setAudience(event.target.value)}><option value="all">All players</option><option value="verified">Verified players</option><option value="restricted">Restricted accounts</option><option value="country">Players with country set</option></Select></div><Input label="Title" value={title} onChange={(event) => setTitle(event.target.value)} minLength={4} maxLength={120} required /><Textarea label="Message" value={message} onChange={(event) => setMessage(event.target.value)} minLength={10} maxLength={4000} required />{error ? <p className="form-error" role="alert">{error}</p> : null}<Button type="submit" disabled={submitting}><Send size={16} />{submitting ? "Sending..." : "Send audited notice"}</Button></form>
        </Panel>
        <Panel title="Delivery history" description="Recent CRM-originated notification broadcasts">
          {resource.loading ? <LoadingState /> : resource.error || !resource.data ? <ErrorState message={resource.error} retry={() => void resource.reload()} /> : resource.data.announcements.length === 0 ? <EmptyState title="No notices sent" description="The CRM has not sent a platform notice." /> : <div className="stack-list">{resource.data.announcements.map((item) => <article key={item.id}><div><strong><BellRing size={15} />{item.title}</strong><Badge>{item.status}</Badge></div><p>{item.message}</p><small>{item.category} · {item.audience} · {dateTime(item.sentAt || item.createdAt)}</small></article>)}</div>}
        </Panel>
      </div>
    </>
  );
}
