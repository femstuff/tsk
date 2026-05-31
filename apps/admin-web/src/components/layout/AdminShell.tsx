import type { ReactNode } from "react";

import type { NavSection } from "../../lib/dashboardAnalytics";

type NavItem = {
  id: NavSection;
  label: string;
  icon: JSX.Element;
};

const NAV_ITEMS: NavItem[] = [
  {
    id: "dashboard",
    label: "Панель управления",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1v-9.5Z" />
      </svg>
    )
  },
  {
    id: "documents",
    label: "Документы",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Z" />
        <path d="M14 3v5h5" />
      </svg>
    )
  },
  {
    id: "tasks",
    label: "Команды",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M9 12.5 11 14.5 15.5 10" />
        <rect x="4" y="4" width="16" height="16" rx="3" />
      </svg>
    )
  },
  {
    id: "users",
    label: "Пользователи",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="8" r="3.5" />
        <path d="M5 19c1.5-3 4.2-4.5 7-4.5s5.5 1.5 7 4.5" />
      </svg>
    )
  },
  {
    id: "templates",
    label: "Шаблоны",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <rect x="4" y="5" width="16" height="5" rx="1.5" />
        <rect x="4" y="12" width="10" height="7" rx="1.5" />
        <rect x="16" y="12" width="4" height="7" rx="1.5" />
      </svg>
    )
  },
  {
    id: "activity",
    label: "Активность",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 14 8 10l4 4 3-3 5 5" />
        <path d="M4 19h16" />
      </svg>
    )
  },
  {
    id: "metrics",
    label: "Метрики",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <rect x="5" y="11" width="3" height="8" rx="1" />
        <rect x="10.5" y="7" width="3" height="12" rx="1" />
        <rect x="16" y="4" width="3" height="15" rx="1" />
      </svg>
    )
  },
  {
    id: "integrations",
    label: "Интеграции",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 3 14.5 9.5 21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5L12 3Z" />
      </svg>
    )
  },
  {
    id: "settings",
    label: "Настройки",
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="3" />
        <path d="M12 2v2M12 20v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M2 12h2M20 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4" />
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
  return (
    <div className="admin-app">
      <aside className="admin-sidebar">
        <div className="sidebar-brand">
          <div className="sidebar-logo">TSK</div>
          <div>
            <strong>Admin</strong>
            <span>Панель управления</span>
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
              <span>{item.label}</span>
            </button>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="sidebar-user">
            <div className="sidebar-avatar">A</div>
            <div>
              <strong>Администратор</strong>
              <span>Локальная среда</span>
            </div>
          </div>
          <button type="button" className="sidebar-logout" onClick={() => void onRefresh()} disabled={loading}>
            {loading ? "Обновление…" : "Обновить"}
          </button>
        </div>
      </aside>

      <div className="admin-main">
        <header className="admin-topbar">
          <div>
            <p className="topbar-eyebrow">TSK Platform</p>
            <h1>{NAV_ITEMS.find((item) => item.id === activeSection)?.label ?? "Панель управления"}</h1>
          </div>
          <button type="button" className="topbar-action" onClick={() => void onRefresh()} disabled={loading}>
            {loading ? "Загрузка…" : "Обновить данные"}
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
