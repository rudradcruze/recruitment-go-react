import { useState } from "react";
import { Link } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { useDeleteQuestion, useQuestions } from "./use-questions";

const buttonClass = "rounded border border-gray-500 px-3 py-1 aria-disabled:opacity-50";

export function QuestionListPage() {
  usePageTitle("Questions");
  const heading = useHeadingFocus();
  const questions = useQuestions();
  const remove = useDeleteQuestion();
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
          Questions
        </h1>
        <Link className="rounded bg-blue-800 px-4 py-2 text-white" to="/admin/questions/new">
          New Question
        </Link>
      </div>
      {/* The regions are always in the page, so a screen reader announces the text that appears in them. */}
      <div className="mt-4 empty:mt-0" role="status">
        {questions.isPending && "The application loads the Questions."}
        {remove.isSuccess && "The Question is deleted."}
      </div>
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {/* A failed refresh keeps the list that the page already shows. */}
        {questions.isError && questions.data === undefined && messageOf(questions.error)}
        {remove.isError && messageOf(remove.error)}
      </div>
      {questions.data?.length === 0 && (
        <p className="mt-6 text-gray-700">
          The Question bank has no Questions. Select &quot;New Question&quot; to add one.
        </p>
      )}
      {questions.data !== undefined && questions.data.length > 0 && (
        <table className="mt-6 w-full text-left">
          <caption className="sr-only">Questions</caption>
          <thead>
            <tr className="border-b border-gray-400">
              <th scope="col" className="py-2">
                Question
              </th>
              <th scope="col" className="py-2">
                Options
              </th>
              <th scope="col" className="py-2">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {questions.data.map((question) => (
              <tr key={question.id} className="border-b border-gray-200">
                <td className="py-2 pr-4">{question.text}</td>
                <td className="py-2 pr-4">{question.options.length}</td>
                <td className="flex flex-wrap gap-2 py-2">
                  <Link className={buttonClass} to={`/admin/questions/${question.id}`}>
                    Edit
                  </Link>
                  {confirming === question.id ? (
                    <>
                      <button
                        className={`${buttonClass} text-red-800`}
                        type="button"
                        aria-disabled={remove.isPending}
                        onClick={() => confirmDelete(question.id)}
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
                      onClick={() => setConfirming(question.id)}
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
