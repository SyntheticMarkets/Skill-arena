"use client";

import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";
import { AlertCircle, CheckCircle2, LoaderCircle, Search, X } from "lucide-react";

export function Button({ children, className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`button ${className}`} {...props}>{children}</button>;
}

export function IconButton({ label, children, className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { label: string; children: ReactNode }) {
  return <button className={`icon-button ${className}`} aria-label={label} title={label} {...props}>{children}</button>;
}

export function Input({ label, hint, ...props }: InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string }) {
  return (
    <label className="field">
      <span>{label}</span>
      <input {...props} />
      {hint ? <small>{hint}</small> : null}
    </label>
  );
}

export function Select({ label, children, ...props }: SelectHTMLAttributes<HTMLSelectElement> & { label: string; children: ReactNode }) {
  return <label className="field"><span>{label}</span><select {...props}>{children}</select></label>;
}

export function Textarea({ label, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string }) {
  return <label className="field"><span>{label}</span><textarea {...props} /></label>;
}

export function SearchInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <label className="search-input"><Search size={16} aria-hidden /><span className="sr-only">Search</span><input aria-label="Search" {...props} /></label>;
}

export function PageHeader({ eyebrow, title, description, actions }: { eyebrow: string; title: string; description: string; actions?: ReactNode }) {
  return (
    <header className="page-header">
      <div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p>{description}</p></div>
      {actions ? <div className="page-actions">{actions}</div> : null}
    </header>
  );
}

export function Panel({ title, description, actions, children, className = "" }: { title: string; description?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`panel ${className}`}>
      <div className="panel-heading"><div><h2>{title}</h2>{description ? <p>{description}</p> : null}</div>{actions}</div>
      {children}
    </section>
  );
}

export function Metric({ label, value, detail, tone = "default" }: { label: string; value: ReactNode; detail?: string; tone?: "default" | "good" | "warning" | "danger" }) {
  return <div className={`metric metric-${tone}`}><p>{label}</p><strong>{value}</strong>{detail ? <small>{detail}</small> : null}</div>;
}

export function Badge({ children, tone }: { children: ReactNode; tone?: string }) {
  const normalized = tone || String(children).toLowerCase().replaceAll(" ", "-");
  return <span className={`badge badge-${normalized}`}>{children}</span>;
}

export function LoadingState({ label = "Loading operational data" }: { label?: string }) {
  return <div className="state"><LoaderCircle className="spin" aria-hidden /><p>{label}</p></div>;
}

export function EmptyState({ title, description }: { title: string; description: string }) {
  return <div className="state"><CheckCircle2 aria-hidden /><h3>{title}</h3><p>{description}</p></div>;
}

export function ErrorState({ message, retry }: { message: string; retry?: () => void }) {
  return <div className="state state-error"><AlertCircle aria-hidden /><h3>Unable to load</h3><p>{message}</p>{retry ? <Button onClick={retry}>Try again</Button> : null}</div>;
}

export function Modal({ title, description, children, onClose }: { title: string; description?: string; children: ReactNode; onClose: () => void }) {
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">
        <div className="panel-heading"><div><h2 id="modal-title">{title}</h2>{description ? <p>{description}</p> : null}</div><IconButton label="Close dialog" onClick={onClose}><X size={18} /></IconButton></div>
        {children}
      </div>
    </div>
  );
}
