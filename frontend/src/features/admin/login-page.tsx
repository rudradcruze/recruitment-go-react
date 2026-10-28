import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";

import { messageOf } from "../../api/errors";
import { usePageTitle } from "../../components/use-page-title";
import { demoAdmin } from "./demo-admin";
import { useLogin, type LoginRequest } from "./use-session";

const fieldClass = "mt-1 w-full rounded border border-gray-500 px-3 py-2";

export function LoginPage() {
  usePageTitle("Log in");
  const [form, setForm] = useState<LoginRequest>({ emailAddress: "", password: "" });
  const login = useLogin();
  const navigate = useNavigate();
  const demo = demoAdmin();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (login.isPending) {
      return;
    }
    login.mutate(form, {
      // Replace the page in the history, so the Back button does not return to the form.
      onSuccess: () => void navigate("/admin", { replace: true }),
    });
  }

  return (
    <main className="mx-auto max-w-md p-8">
      <h1 className="text-3xl font-bold">Log in</h1>
      {/* The server checks the values and gives the messages, so the browser does not check them. */}
      <form className="mt-6 flex flex-col gap-4" onSubmit={submit} noValidate>
        <div>
          <label htmlFor="emailAddress">Email address</label>
          <input
            id="emailAddress"
            className={fieldClass}
            type="email"
            autoComplete="username"
            value={form.emailAddress}
            onChange={(event) => setForm({ ...form, emailAddress: event.target.value })}
          />
        </div>
        <div>
          <label htmlFor="password">Password</label>
          <input
            id="password"
            className={fieldClass}
            type="password"
            autoComplete="current-password"
            value={form.password}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
          />
        </div>
        {/* The button uses aria-disabled, not disabled: a disabled button loses the keyboard focus. submit ignores a second send. */}
        <button
          className="rounded bg-blue-800 px-4 py-2 text-white aria-disabled:opacity-50"
          type="submit"
          aria-disabled={login.isPending}
        >
          Log in
        </button>
      </form>
      {/* The region is always in the page. A screen reader announces the text that appears in a region that was there before. */}
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {login.isError && messageOf(login.error)}
      </div>
      {demo !== undefined && (
        <section
          aria-labelledby="demo-admin-heading"
          className="mt-8 rounded border border-gray-300 p-4"
        >
          <h2 id="demo-admin-heading" className="font-bold">
            Demo Admin
          </h2>
          <p className="mt-1 text-gray-700">Use this account to try the application.</p>
          <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
            <dt>Email address</dt>
            <dd>
              <code>{demo.emailAddress}</code>
            </dd>
            <dt>Password</dt>
            <dd>
              <code>{demo.password}</code>
            </dd>
          </dl>
        </section>
      )}
    </main>
  );
}
