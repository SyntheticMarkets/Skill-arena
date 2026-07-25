"use client";

import { FormEvent, useState } from "react";
import { LockKeyhole, MessageSquareReply, Send } from "lucide-react";
import { api, apiURL, APIError, dateTime } from "@/lib/api";
import type { SupportTicket } from "@/lib/types";
import { useResource } from "@/lib/use-resource";
import { Badge, Button, EmptyState, ErrorState, LoadingState, Modal, PageHeader, Panel, Select, Textarea } from "@/components/ui";

export default function SupportPage() {
  const [status, setStatus] = useState("");
  const resource = useResource<{ tickets: SupportTicket[] }>(`/api/v1/admin-crm/support/tickets?status=${status}`);
  const [selected, setSelected] = useState<SupportTicket | null>(null);
  return (
    <>
      <PageHeader eyebrow="Support CRM" title="Player support" description="Assign, reply, escalate, and close player conversations. Internal notes never appear in player channels." />
      <Panel title="Support queue" description="Ordered by latest activity" actions={<select aria-label="Ticket status" value={status} onChange={(event) => setStatus(event.target.value)}><option value="">All tickets</option><option value="open">Open</option><option value="in_progress">In progress</option><option value="waiting_player">Waiting for player</option><option value="escalated">Escalated</option><option value="closed">Closed</option></select>}>
        {resource.loading ? <LoadingState /> : resource.error || !resource.data ? <ErrorState message={resource.error} retry={() => void resource.reload()} /> : resource.data.tickets.length === 0 ? <EmptyState title="Queue clear" description="No support tickets match this status." /> : <div className="ticket-list">{resource.data.tickets.map((ticket) => <button key={ticket.id} onClick={() => setSelected(ticket)}><div className={`priority-marker priority-${ticket.priority}`} /><div><div><strong>{ticket.subject}</strong><Badge>{ticket.status}</Badge>{ticket.escalated ? <Badge tone="danger">Escalated</Badge> : null}</div><p>{ticket.category.replaceAll("_", " ")} · {ticket.userId}</p></div><div><span>{ticket.messages.length} messages</span><time>{dateTime(ticket.updatedAt)}</time></div></button>)}</div>}
      </Panel>
      {selected ? <TicketModal ticket={selected} close={() => setSelected(null)} complete={async () => { setSelected(null); await resource.reload(); }} /> : null}
    </>
  );
}

function TicketModal({ ticket, close, complete }: { ticket: SupportTicket; close: () => void; complete: () => Promise<void> }) {
  const [status, setStatus] = useState(ticket.status);
  const [priority, setPriority] = useState(ticket.priority);
  const [assignedTo, setAssignedTo] = useState(ticket.assignedTo || "");
  const [escalated, setEscalated] = useState(ticket.escalated);
  const [reply, setReply] = useState("");
  const [internal, setInternal] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      await api(`/api/v1/admin-crm/support/tickets/${ticket.id}`, { method: "PATCH", body: JSON.stringify({ status, priority, assignedTo, escalated, reply, internal }) });
      await complete();
    } catch (requestError) {
      setError(requestError instanceof APIError ? requestError.message : "The support update could not be saved.");
    }
  }
  return <Modal title={ticket.subject} description={`${ticket.category.replaceAll("_", " ")} / ${ticket.userId}`} onClose={close}>
    <div className="support-conversation">{ticket.messages.map((message) => <article key={message.id} className={message.internal ? "internal-message" : ""}><div>{message.internal ? <LockKeyhole size={14} /> : <MessageSquareReply size={14} />}<strong>{message.authorId}</strong><time>{dateTime(message.createdAt)}</time></div><p>{message.body}</p>{message.internal ? <Badge tone="warning">Internal only</Badge> : null}</article>)}</div>
    {ticket.attachments?.length ? <section className="record-section"><h3>Attachments</h3><div className="stack-list">{ticket.attachments.map((attachment) => <a className="evidence-row" key={attachment.id} href={apiURL(`/api/v1/admin-crm/support/attachments/${attachment.id}`)} target="_blank" rel="noreferrer"><div><strong>{attachment.fileName}</strong><span>{attachment.contentType} / {Math.ceil(attachment.sizeBytes / 1024)} KB</span></div><Badge>{attachment.sha256.slice(0, 10)}</Badge></a>)}</div></section> : null}
    <form className="stack" onSubmit={submit}><div className="form-grid"><Select label="Status" value={status} onChange={(event) => setStatus(event.target.value)}><option value="open">Open</option><option value="received">Received</option><option value="in_progress">In progress</option><option value="waiting_player">Waiting for player</option><option value="escalated">Escalated</option><option value="closed">Closed</option></Select><Select label="Priority" value={priority} onChange={(event) => setPriority(event.target.value)}><option value="low">Low</option><option value="normal">Normal</option><option value="high">High</option><option value="urgent">Urgent</option></Select><label className="field"><span>Assigned administrator ID</span><input value={assignedTo} onChange={(event) => setAssignedTo(event.target.value)} placeholder="Unassigned" /></label></div><Textarea label={internal ? "Internal note" : "Player reply"} value={reply} onChange={(event) => setReply(event.target.value)} maxLength={8000} /><div className="check-grid"><label><input type="checkbox" checked={internal} onChange={(event) => setInternal(event.target.checked)} />Internal only</label><label><input type="checkbox" checked={escalated} onChange={(event) => setEscalated(event.target.checked)} />Escalated</label></div>{error ? <p className="form-error">{error}</p> : null}<div className="button-row"><Button type="button" className="button-secondary" onClick={close}>Cancel</Button><Button type="submit"><Send size={16} />Save update</Button></div></form>
  </Modal>;
}
