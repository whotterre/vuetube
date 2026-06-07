import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { Field } from "@/components/Field";

export const Route = createFileRoute("/signup")({
  component: SignupPage,
});

function SignupPage() {
  const { signup } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ first_name: "", last_name: "", email: "", password: "", country: ""});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const update = (k: keyof typeof form) => (v: string) => setForm((f) => ({ ...f, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await signup(form);
      navigate({ to: "/" });
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="mx-auto flex max-w-md flex-col px-6 py-16">
      <h1 className="text-3xl font-bold tracking-tight">Create your account</h1>
      <p className="mt-1 text-sm text-muted-foreground">Join VueTube and start sharing.</p>
      <form onSubmit={submit} className="mt-8 space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <Field label="First name" type="text" value={form.first_name} onChange={update("first_name")} />
          <Field label="Last name" type="text" value={form.last_name} onChange={update("last_name")} />
        </div>
        <Field label="Email" type="email" value={form.email} onChange={update("email")} />
        <Field label="Password" type="password" value={form.password} onChange={update("password")} />
        <Field label="Country">
          <select
            required
            value={form.country}
            onChange={(e) => update("country")(e.target.value)}
            className="mt-2 w-full rounded-md border border-border bg-input px-3 py-2.5 text-sm outline-none focus:border-primary"
          >
            <option value="" disabled>Select your country</option>
            <option value="US">United States</option>
            <option value="UK">United Kingdom</option>
            <option value="CA">Canada</option>
            <option value="AU">Australia</option>
            <option value="FR">France</option>
            <option value="DE">Germany</option>
            <option value="JP">Japan</option>
            <option value="IN">India</option>
            <option value="BR">Brazil</option>
            <option value="OTHER">Other</option>
          </select>
        </Field>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <button
          disabled={loading}
          className="w-full rounded-md bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50"
        >
          {loading ? "Creating…" : "Create account"}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-muted-foreground">
        Already have an account? <Link to="/login" className="text-primary hover:underline">Sign in</Link>
      </p>
    </main>
  );
}
