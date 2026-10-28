import { useParams } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { formatTime } from "./datetime";
import { useExam, useExamAttempts, type AttemptResult } from "./use-exams";

const statusText: Record<AttemptResult["status"], string> = {
  open: "Open",
  submitted: "Submitted",
  timed_out: "Timed out",
};

export function ExamResultsPage() {
  usePageTitle("Results");
  const { id = "" } = useParams();
  const heading = useHeadingFocus();
  const exam = useExam(id);
  const attempts = useExamAttempts(id);

  return (
    <main className="mx-auto max-w-4xl p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        {exam.data === undefined ? "Results" : `Results of ${exam.data.title}`}
      </h1>
      {/* The regions are always in the page, so a screen reader announces the text that appears in them. */}
      <div className="mt-4 empty:mt-0" role="status">
        {attempts.isPending && "The application loads the results."}
      </div>
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {attempts.isError && attempts.data === undefined && messageOf(attempts.error)}
      </div>
      {attempts.data?.length === 0 && (
        <p className="mt-6 text-gray-700">No Candidate started this Exam.</p>
      )}
      {attempts.data !== undefined && attempts.data.length > 0 && (
        <table className="mt-6 w-full text-left">
          <caption className="sr-only">Results</caption>
          <thead>
            <tr className="border-b border-gray-400">
              <th scope="col" className="py-2">
                Name
              </th>
              <th scope="col" className="py-2">
                Email address
              </th>
              <th scope="col" className="py-2">
                Status
              </th>
              <th scope="col" className="py-2">
                Score
              </th>
              <th scope="col" className="py-2">
                Started
              </th>
              <th scope="col" className="py-2">
                Finished
              </th>
            </tr>
          </thead>
          <tbody>
            {attempts.data.map((attempt) => (
              <tr
                key={`${attempt.emailAddress}-${attempt.startedAt}`}
                className="border-b border-gray-200"
              >
                <td className="py-2 pr-4">{attempt.name}</td>
                <td className="py-2 pr-4">{attempt.emailAddress}</td>
                <td className="py-2 pr-4">{statusText[attempt.status]}</td>
                <td className="py-2 pr-4">
                  {attempt.score === null
                    ? "Not available"
                    : `${attempt.score.correct} of ${attempt.score.total}`}
                </td>
                <td className="py-2 pr-4">{formatTime(attempt.startedAt)}</td>
                <td className="py-2 pr-4">
                  {attempt.finishedAt === null ? "Not finished" : formatTime(attempt.finishedAt)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
