import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import {
  useAttempt,
  useSaveAnswer,
  useSubmitAttempt,
  type Attempt,
  type Score,
} from "./use-attempt";

// AttemptPage is the page where a Candidate answers the Questions. The Attempt ID in the address is the secret of the Candidate.
export function AttemptPage() {
  const { id = "" } = useParams();
  const attempt = useAttempt(id);
  usePageTitle(attempt.data?.examTitle ?? "Exam");
  const heading = useHeadingFocus();

  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        {attempt.data?.examTitle ?? "Exam"}
      </h1>
      {attempt.data === undefined && (
        <>
          <div className="mt-4 empty:mt-0" role="status">
            {attempt.isPending && "The application loads the Exam."}
          </div>
          <div className="mt-4 text-red-800 empty:mt-0" role="alert">
            {attempt.isError && messageOf(attempt.error)}
          </div>
        </>
      )}
      {/* The API gives only the status for a submitted or timed-out Attempt, without the Score and the Questions. */}
      {attempt.data !== undefined && attempt.data.status !== "open" && (
        <p className="mt-6">This Exam is already submitted.</p>
      )}
      {attempt.data !== undefined && attempt.data.status === "open" && (
        <OpenAttempt key={id} id={id} attempt={attempt.data} />
      )}
    </main>
  );
}

function secondsLeft(expiresAt: string): number {
  return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 1000));
}

function formatCountdown(seconds: number): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(Math.floor(seconds / 60))}:${pad(seconds % 60)}`;
}

function OpenAttempt({ id, attempt }: { id: string; attempt: Attempt }) {
  const [answers, setAnswers] = useState<Record<string, string>>(() =>
    Object.fromEntries(
      (attempt.answers ?? []).map((answer) => [answer.questionId, answer.optionId]),
    ),
  );
  const [score, setScore] = useState<Score | undefined>();
  const [remaining, setRemaining] = useState(() => secondsLeft(attempt.expiresAt ?? ""));
  const save = useSaveAnswer(id);
  const submit = useSubmitAttempt(id);
  const autoSubmitted = useRef(false);

  function submitAttempt() {
    if (submit.isPending || score !== undefined) {
      return;
    }
    submit.mutate(undefined, { onSuccess: setScore });
  }

  // The countdown shows the time to the expiry. The server has its own check of the Time limit.
  useEffect(() => {
    if (score !== undefined) {
      return;
    }
    const timer = setInterval(() => setRemaining(secondsLeft(attempt.expiresAt ?? "")), 1000);
    return () => clearInterval(timer);
  }, [attempt.expiresAt, score]);

  // When the Time limit is over, submit one time. The server accepts a Submit after the expiry.
  useEffect(() => {
    if (remaining === 0 && !autoSubmitted.current && score === undefined) {
      autoSubmitted.current = true;
      submit.mutate(undefined, { onSuccess: setScore });
    }
  }, [remaining, score, submit]);

  function select(questionId: string, optionId: string) {
    const previous = answers[questionId];
    setAnswers({ ...answers, [questionId]: optionId });
    save.mutate(
      { questionId, optionId },
      {
        // The server did not save the Answer, so the page shows the Answer that the server has.
        onError: () =>
          setAnswers((current) => {
            const { [questionId]: _removed, ...others } = current;
            return previous === undefined ? others : { ...others, [questionId]: previous };
          }),
      },
    );
  }

  return (
    <>
      {score === undefined && (
        <p className="mt-4 text-lg">
          Time left: <span role="timer">{formatCountdown(remaining)}</span>
        </p>
      )}
      {/* The regions are always in the page, so a screen reader announces the text that appears in them. */}
      <div className="mt-4 empty:mt-0" role="status">
        {score === undefined && save.isSuccess && "The Answer is saved."}
        {score !== undefined && `Your Score is ${score.correct} of ${score.total}.`}
      </div>
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {save.isError && messageOf(save.error)}
        {submit.isError && messageOf(submit.error)}
      </div>
      {score === undefined && (
        <form
          className="mt-6 flex flex-col gap-6"
          onSubmit={(event) => {
            event.preventDefault();
            submitAttempt();
          }}
        >
          {(attempt.questions ?? []).map((question) => (
            <fieldset key={question.id} className="flex flex-col gap-2">
              <legend className="mb-2 font-bold">{question.text}</legend>
              {question.options.map((option) => (
                <label key={option.id} className="flex items-start gap-2">
                  <input
                    type="radio"
                    name={question.id}
                    checked={answers[question.id] === option.id}
                    onChange={() => select(question.id, option.id)}
                  />
                  {option.text}
                </label>
              ))}
            </fieldset>
          ))}
          {/* The button uses aria-disabled, not disabled: a disabled button loses the keyboard focus. submitAttempt ignores a second send. */}
          <button
            className="self-start rounded bg-blue-800 px-4 py-2 text-white aria-disabled:opacity-50"
            type="submit"
            aria-disabled={submit.isPending}
          >
            Submit
          </button>
        </form>
      )}
    </>
  );
}
