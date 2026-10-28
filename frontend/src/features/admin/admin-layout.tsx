import { Link, Navigate, Outlet, useNavigate } from "react-router";

import { ApiError, messageOf } from "../../api/errors";
import { useLogout, useSession } from "./use-session";

// AdminLayout guards the Admin pages. A person without a session goes to the login page.
export function AdminLayout() {
  const session = useSession();
  const logout = useLogout();
  const navigate = useNavigate();

  // The session ended, also after the page already had it. The API answers 401 to the next request.
  if (session.error instanceof ApiError && session.error.status === 401) {
    return <Navigate to="/admin/login" replace />;
  }
  if (session.data === undefined) {
    return (
      <main className="mx-auto max-w-4xl p-8">
        {/* The 2 regions are in the page from the first render, so a screen reader announces their text. */}
        <div role="status">{session.isPending && "The application checks your session."}</div>
        <div className="text-red-800" role="alert">
          {session.isError && messageOf(session.error)}
        </div>
      </main>
    );
  }

  function logOut() {
    if (logout.isPending) {
      return;
    }
    logout.mutate(undefined, { onSuccess: () => void navigate("/admin/login", { replace: true }) });
  }

  return (
    <div>
      <header className="flex flex-wrap items-center gap-4 border-b border-gray-300 px-8 py-3">
        <nav aria-label="Admin menu" className="flex gap-4">
          <Link className="underline" to="/admin/questions">
            Questions
          </Link>
          <Link className="underline" to="/admin/exams">
            Exams
          </Link>
        </nav>
        <p className="mr-auto">
          Logged in as <strong>{session.data.emailAddress}</strong>
        </p>
        <button
          className="rounded border border-gray-500 px-3 py-1 aria-disabled:opacity-50"
          type="button"
          onClick={logOut}
          aria-disabled={logout.isPending}
        >
          Log out
        </button>
      </header>
      <div className="px-8 text-red-800" role="alert">
        {logout.isError && messageOf(logout.error)}
      </div>
      <Outlet />
    </div>
  );
}
