const variantStyles = {
  default: {
    wrapper: "bg-bg-elevated border-border-subtle",
    icon: "text-txt-muted",
    text: "text-txt-primary",
  },
  fuel: {
    wrapper: "bg-amber-400/10 border-amber-400/30",
    icon: "text-amber-400",
    text: "text-amber-300",
  },
  drive: {
    wrapper: "bg-accent/10 border-accent/30",
    icon: "text-accent",
    text: "text-accent",
  },
  engine: {
    wrapper: "bg-sky-500/10 border-sky-500/30",
    icon: "text-sky-400",
    text: "text-sky-300",
  },
};

export function HeroBadge({ label, variant = "default", icon: Icon }) {
  if (!label) return null;
  const s = variantStyles[variant] ?? variantStyles.default;
  return (
    <span
      className={`inline-flex items-center gap-2 px-4 py-2 rounded-full border text-sm font-medium ${s.wrapper}`}
    >
      {Icon && (
        <Icon size={15} strokeWidth={2} className={`shrink-0 ${s.icon}`} />
      )}
      <span className={s.text}>{label}</span>
    </span>
  );
}
