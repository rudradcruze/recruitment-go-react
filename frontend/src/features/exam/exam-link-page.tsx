import { useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { formatTime } from "./datetime";
import { useExamLink, useStartAttempt, type StartAttemptRequest } from "./use-attempt";

const fieldClass = "mt-1 w-full rounded border border-gray-500 px-3 py-2";

// ExamLinkPage is the page of an Exam link. A Candidate gives a name and an email address, and starts the Exam.
export function ExamLinkPage() {
  const { token = "" } = useParams();
  const examLink = useExamLink(token);
  const start = useStartAttempt(token);
  const navigate = useNavigate();
  const [form, setForm] = useState<StartAttemptRequest>({ name: "", emailAddress: "" });
  usePageTitle(examLink.data?.title ?? "Exam");
  const heading = useHeadingFocus();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (start.isPending) {
      return;
    }
    start.mutate(form, {
      // Replace the page in the history, so the Back button does not return to the form.
      onSuccess: (started) => void navigate(`/attempts/${started.id}`, { replace: true }),
    });
  }

  return (
    <main className="mx-auto max-w-md p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        {examLink.data?.title ?? "Exam"}
      </h1>
      {examLink.data === undefined && (
        <>
          <div className="mt-4 empty:mt-0" role="status">
            {examLink.isPending && "The application loads the Exam."}
          </div>
          <div className="mt-4 text-red-800 empty:mt-0" role="alert">
            {examLink.isError && messageOf(examLink.error)}
          </div>
        </>
      )}
      {examLink.data !== undefined && (
        <>
          <dl className="mt-6 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
            <dt className="font-bold">Opens</dt>
            <dd>{formatTime(examLink.data.opensAt)}</dd>
            <dt className="font-bold">Closes</dt>
            <dd>{formatTime(examLink.data.closesAt)}</dd>
            <dt className="font-bold">Time limit</dt>
            <dd>{examLink.data.timeLimitMinutes} minutes</dd>
            <dt className="font-bold">Questions</dt>
            <dd>{examLink.data.questionCount}</dd>
          </dl>
          {/* The server checks the values and gives the messages, so the browser does not check them. */}
          <form className="mt-6 flex flex-col gap-4" onSubmit={submit} noValidate>
            <div>
              <label htmlFor="name">Name</label>
              <input
                id="name"
                className={fieldClass}
                type="text"
                autoComplete="name"
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
              />
            </div>
            <div>
              <label htmlFor="emailAddress">Email address</label>
              <input
                id="emailAddress"
                className={fieldClass}
                type="email"
                autoComplete="email"
                value={form.emailAddress}
                onChange={(event) => setForm({ ...form, emailAddress: event.target.value })}
              />
            </div>
            {/* The button uses aria-disabled, not disabled: a disabled button loses the keyboard focus. submit ignores a second send. */}
            <button
              className="rounded bg-blue-800 px-4 py-2 text-white aria-disabled:opacity-50"
              type="submit"
              aria-disabled={start.isPending}
            >
              Start
            </button>
          </form>
          {/* The region is always in the page. A screen reader announces the text that appears in a region that was there before. */}
          <div className="mt-4 text-red-800 empty:mt-0" role="alert">
            {start.isError && messageOf(start.error)}
          </div>
        </>
      )}
    </main>
  );
}
