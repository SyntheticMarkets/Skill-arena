"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity, Bell, BookOpenCheck, ChevronLeft, ChevronRight, CircleDollarSign,
  FileClock, Headphones, LayoutDashboard, LogOut, Menu, ShieldCheck, Users, X
} from "lucide-react";
import { useState } from "react";
import { useAuth } from "@/app/auth-provider";
import { IconButton, LoadingState } from "./ui";

const navigation = [
  { href: "/", label: "Operations", icon: LayoutDashboard, permission: "dashboard.read" },
  { href: "/users", label: "Players", icon: Users, permission: "users.read" },
  { href: "/finance", label: "Finance", icon: CircleDollarSign, permission: "finance.read" },
  { href: "/compliance", label: "Compliance", icon: ShieldCheck, permission: "compliance.read" },
  { href: "/support", label: "Support", icon: Headphones, permission: "support.read" },
  { href: "/audit", label: "Audit", icon: FileClock, permission: "audit.read" },
  { href: "/announcements", label: "Notices", icon: Bell, permission: "notifications.send" },
  { href: "/monitoring", label: "Monitoring", icon: Activity, permission: "monitoring.read" }
];

export function AdminShell({ children }: { children: React.ReactNode }) {
  const { admin, loading, logout, can } = useAuth();
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  if (loading || !admin) return <main className="centered"><LoadingState label="Verifying administrator session" /></main>;

  return (
    <div className={`crm-shell ${collapsed ? "shell-collapsed" : ""}`}>
      <aside className={`sidebar ${mobileOpen ? "sidebar-open" : ""}`}>
        <div className="brand">
          <div className="brand-mark"><BookOpenCheck size={20} aria-hidden /></div>
          <div><strong>Skill Arena</strong><span>Operations</span></div>
          <IconButton label="Close navigation" className="mobile-close" onClick={() => setMobileOpen(false)}><X size={18} /></IconButton>
        </div>
        <nav aria-label="Operations navigation">
          {navigation.filter((item) => can(item.permission)).map((item) => {
            const active = item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
            return <Link key={item.href} href={item.href} className={active ? "active" : ""} onClick={() => setMobileOpen(false)} title={collapsed ? item.label : undefined}><item.icon size={18} aria-hidden /><span>{item.label}</span></Link>;
          })}
        </nav>
        <div className="sidebar-footer">
          <button className="collapse-control" onClick={() => setCollapsed((value) => !value)} title={collapsed ? "Expand navigation" : "Collapse navigation"}>
            {collapsed ? <ChevronRight size={17} /> : <ChevronLeft size={17} />}<span>{collapsed ? "Expand" : "Collapse"}</span>
          </button>
          <button onClick={() => void logout()}><LogOut size={18} aria-hidden /><span>Sign out</span></button>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <IconButton label="Open navigation" className="mobile-menu" onClick={() => setMobileOpen(true)}><Menu size={20} /></IconButton>
          <div className="environment"><span />Production operations</div>
          <div className="operator"><div><strong>{admin.displayName || admin.email}</strong><span>{admin.role.replaceAll("_", " ")}</span></div><div className="operator-avatar">{(admin.displayName || admin.email).slice(0, 2).toUpperCase()}</div></div>
        </header>
        <main className="content">{children}</main>
      </div>
      {mobileOpen ? <button className="sidebar-scrim" aria-label="Close navigation" onClick={() => setMobileOpen(false)} /> : null}
    </div>
  );
}
