import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";

export function AdminHomePage() {
  usePageTitle("Admin");
  const heading = useHeadingFocus();

  return (
    <main className="mx-auto max-w-4xl p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        Admin
      </h1>
      <p className="mt-4 text-gray-700">You are logged in.</p>
    </main>
  );
}
