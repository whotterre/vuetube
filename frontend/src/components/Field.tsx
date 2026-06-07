export function Field({ 
  label, 
  type, 
  value, 
  onChange,
  children,
}: { 
  label: string; 
  type?: string; 
  value?: string; 
  onChange?: (v: string) => void;
  children?: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium">{label}</span>
      {children ? (
        children
      ) : (
        <input
          type={type}
          required
          value={value}
          onChange={(e) => onChange?.(e.target.value)}
          className="mt-2 w-full rounded-md border border-border bg-input px-3 py-2.5 text-sm outline-none focus:border-primary"
        />
      )}
    </label>
  );
}
