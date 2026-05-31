import type { ReactNode } from "react";

import type { NavSection } from "../../lib/dashboardAnalytics";

type NavItem = {
  id: NavSection;
  label: string;
  description: string;
  icon: JSX.Element;
};

const NAV_ITEMS: NavItem[] = [
  {
    id: "overview",
    label: "Обзор",
    description: "Сводка платформы",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1v-9.5Z" />
      </svg>
    )
  },
  {
    id: "bitrix",
    label: "Bitrix24",
    description: "Пользователи и задачи",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="8" r="3.5" />
        <path d="M5 19c1.5-3 4.2-4.5 7-4.5s5.5 1.5 7 4.5" />
      </svg>
    )
  },
  {
    id: "jobs",
    label: "Документы",
    description: "Заявки и шаблоны",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Z" />
        <path d="M14 3v5h5" />
      </svg>
    )
  },
  {
    id: "events",
    label: "Журнал",
    description: "События и команды",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 14 8 10l4 4 3-3 5 5" />
        <path d="M4 19h16" />
      </svg>
    )
  },
  {
    id: "health",
    label: "Сервер",
    description: "Состояние и метрики",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <rect x="5" y="11" width="3" height="8" rx="1" />
        <rect x="10.5" y="7" width="3" height="12" rx="1" />
        <rect x="16" y="4" width="3" height="15" rx="1" />
      </svg>
    )
  }
];

type AdminShellProps = {
  activeSection: NavSection;
  onNavigate: (section: NavSection) => void;
  onRefresh: () => void;
  loading: boolean;
  children: ReactNode;
  integrations: Array<{ name: string; status: string; tone: "ok" | "warn" | "muted" }>;
};

export function AdminShell({
  activeSection,
  onNavigate,
  onRefresh,
  loading,
  children,
  integrations
}: AdminShellProps) {
  const activeItem = NAV_ITEMS.find((item) => item.id === activeSection);

  return (
    <div className="admin-app">
      <aside className="admin-sidebar">
        <div className="sidebar-brand">
          <div className="sidebar-logo">TSK</div>
          <div>
            <strong>Admin</strong>
            <span>Платформа документов</span>
          </div>
        </div>

        <nav className="sidebar-nav">
          {NAV_ITEMS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={`sidebar-link${activeSection === item.id ? " active" : ""}`}
              onClick={() => onNavigate(item.id)}
            >
              {item.icon}
              <span className="sidebar-link-text">
                <strong>{item.label}</strong>
                <small>{item.description}</small>
              </span>
            </button>
          ))}
        </nav>

        <div className="sidebar-footer">
          <button type="button" className="sidebar-logout" onClick={() => void onRefresh()} disabled={loading}>
            {loading ? "Обновление…" : "Обновить данные"}
          </button>
        </div>
      </aside>

      <div className="admin-main">
        <header className="admin-topbar">
          <div>
            <p className="topbar-eyebrow">TSK Platform</p>
            <h1>{activeItem?.label ?? "Обзор"}</h1>
            {activeItem?.description ? <p className="topbar-subtitle">{activeItem.description}</p> : null}
          </div>
          <button type="button" className="topbar-action" onClick={() => void onRefresh()} disabled={loading}>
            {loading ? "Загрузка…" : "Обновить"}
          </button>
        </header>

        <div className="admin-content">{children}</div>

        <footer className="admin-integrations">
          {integrations.map((item) => (
            <div key={item.name} className={`integration-pill tone-${item.tone}`}>
              <span>{item.name}</span>
              <strong>{item.status}</strong>
            </div>
          ))}
        </footer>
      </div>
    </div>
  );
}
