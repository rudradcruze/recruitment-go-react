import { useState } from "react";
import { Link } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { formatTime } from "./datetime";
import { useDeleteExam, useExams } from "./use-exams";

const buttonClass = "rounded border border-gray-500 px-3 py-1 aria-disabled:opacity-50";

export function ExamListPage() {
  usePageTitle("Exams");
  const heading = useHeadingFocus();
  const exams = useExams();
  const remove = useDeleteExam();
  const [confirming, setConfirming] = useState<string | undefined>();

  function confirmDelete(id: string) {
    if (remove.isPending) {
      return;
    }
    remove.mutate(id, { onSettled: () => setConfirming(undefined) });
  }

  return (
    <main className="mx-auto max-w-4xl p-8">
      <div className="flex items-center justify-between">
        <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
          Exams
        </h1>
        <Link className="rounded bg-blue-800 px-4 py-2 text-white" to="/admin/exams/new">
          New Exam
        </Link>
      </div>
      {/* The regions are always in the page, so a screen reader announces the text that appears in them. */}
      <div className="mt-4 empty:mt-0" role="status">
        {exams.isPending && "The application loads the Exams."}
        {remove.isSuccess && "The Exam is deleted."}
      </div>
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {/* A failed refresh keeps the list that the page already shows. */}
        {exams.isError && exams.data === undefined && messageOf(exams.error)}
        {remove.isError && messageOf(remove.error)}
      </div>
      {exams.data?.length === 0 && (
        <p className="mt-6 text-gray-700">
          There is no Exam. Select &quot;New Exam&quot; to schedule one.
        </p>
      )}
      {exams.data !== undefined && exams.data.length > 0 && (
        <table className="mt-6 w-full text-left">
          <caption className="sr-only">Exams</caption>
          <thead>
            <tr className="border-b border-gray-400">
              <th scope="col" className="py-2">
                Title
              </th>
              <th scope="col" className="py-2">
                Opens
              </th>
              <th scope="col" className="py-2">
                Closes
              </th>
              <th scope="col" className="py-2">
                Time limit
              </th>
              <th scope="col" className="py-2">
                Questions
              </th>
              <th scope="col" className="py-2">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {exams.data.map((exam) => (
              <tr key={exam.id} className="border-b border-gray-200">
                <td className="py-2 pr-4">{exam.title}</td>
                <td className="py-2 pr-4">{formatTime(exam.opensAt)}</td>
                <td className="py-2 pr-4">{formatTime(exam.closesAt)}</td>
                <td className="py-2 pr-4">{exam.timeLimitMinutes} minutes</td>
                <td className="py-2 pr-4">{exam.questionCount}</td>
                <td className="flex flex-wrap gap-2 py-2">
                  <Link className={buttonClass} to={`/admin/exams/${exam.id}`}>
                    Edit
                  </Link>
                  <Link className={buttonClass} to={`/admin/exams/${exam.id}/results`}>
                    Results
                  </Link>
                  {confirming === exam.id ? (
                    <>
                      <button
                        className={`${buttonClass} text-red-800`}
                        type="button"
                        aria-disabled={remove.isPending}
                        onClick={() => confirmDelete(exam.id)}
                      >
                        Confirm delete
                      </button>
                      <button
                        className={buttonClass}
                        type="button"
                        onClick={() => setConfirming(undefined)}
                      >
                        Cancel
                      </button>
                    </>
                  ) : (
                    <button
                      className={buttonClass}
                      type="button"
                      onClick={() => setConfirming(exam.id)}
                    >
                      Delete
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
