type SectionTabItem<T extends string> = {
  id: T;
  label: string;
  count?: number;
};

type SectionTabsProps<T extends string> = {
  items: SectionTabItem<T>[];
  active: T;
  onChange: (id: T) => void;
};

export function SectionTabs<T extends string>({ items, active, onChange }: SectionTabsProps<T>) {
  return (
    <div className="section-tabs" role="tablist">
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          role="tab"
          aria-selected={active === item.id}
          className={`section-tab${active === item.id ? " active" : ""}`}
          onClick={() => onChange(item.id)}
        >
          {item.label}
          {item.count !== undefined ? <span className="section-tab-count">{item.count}</span> : null}
        </button>
      ))}
    </div>
  );
}
