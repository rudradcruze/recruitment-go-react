import { usePageTitle } from "../components/use-page-title";

export function HomePage() {
  usePageTitle("Home");

  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">go-react-template</h1>
      <p className="mt-4 text-gray-700">This template has a Go API and a React frontend.</p>
    </main>
  );
}
