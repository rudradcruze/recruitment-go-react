import { useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { fromLocalInput, toLocalInput } from "./datetime";
import { useExam, useRegenerateExamLink, useSaveExam, type Exam } from "./use-exams";
import { useQuestions, type Question } from "./use-questions";

const fieldClass = "mt-1 w-full rounded border border-gray-500 px-3 py-2";
const buttonClass = "rounded border border-gray-500 px-3 py-1 aria-disabled:opacity-50";

interface FormState {
  title: string;
  questionIds: string[]; // In the order that the Admin selected them.
  opensAt: string; // The value of a datetime-local field.
  closesAt: string;
  timeLimit: string; // The value of a number field.
}

function stateOf(exam: Exam | undefined): FormState {
  if (exam === undefined) {
    return { title: "", questionIds: [], opensAt: "", closesAt: "", timeLimit: "60" };
  }
  return {
    title: exam.title,
    questionIds: exam.questionIds,
    opensAt: toLocalInput(exam.opensAt),
    closesAt: toLocalInput(exam.closesAt),
    timeLimit: String(exam.timeLimitMinutes),
  };
}

// ExamFormPage makes a new Exam, or changes the Exam that the address names.
export function ExamFormPage() {
  const { id } = useParams();
  const exam = useExam(id);
  const questions = useQuestions();
  usePageTitle(id === undefined ? "New Exam" : "Edit Exam");
  const heading = useHeadingFocus();

  const ready = questions.data !== undefined && (id === undefined || exam.data !== undefined);
  const failure =
    (id !== undefined && exam.data === undefined ? exam.error : null) ??
    (questions.data === undefined ? questions.error : null);

  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        {id === undefined ? "New Exam" : "Edit Exam"}
      </h1>
      {!ready && (
        <>
          <div className="mt-4 empty:mt-0" role="status">
            {failure === null && "The application loads the data."}
          </div>
          <div className="mt-4 text-red-800 empty:mt-0" role="alert">
            {failure !== null && messageOf(failure)}
          </div>
        </>
      )}
      {ready && (
        <>
          <ExamForm
            key={exam.data?.id ?? "new"}
            examId={id}
            initial={stateOf(exam.data)}
            questions={questions.data}
          />
          {exam.data !== undefined && <ExamLinkPanel exam={exam.data} />}
        </>
      )}
    </main>
  );
}

function ExamForm({
  examId,
  initial,
  questions,
}: {
  examId: string | undefined;
  initial: FormState;
  questions: Question[];
}) {
  const [form, setForm] = useState<FormState>(initial);
  const save = useSaveExam(examId);
  const navigate = useNavigate();

  function toggle(questionId: string) {
    const selected = form.questionIds.includes(questionId);
    setForm({
      ...form,
      questionIds: selected
        ? form.questionIds.filter((id) => id !== questionId)
        : [...form.questionIds, questionId],
    });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (save.isPending) {
      return;
    }
    const body = {
      title: form.title,
      questionIds: form.questionIds,
      opensAt: fromLocalInput(form.opensAt),
      closesAt: fromLocalInput(form.closesAt),
      timeLimitMinutes: Number(form.timeLimit),
    };
    // Replace the page in the history, so the Back button does not return to the form.
    save.mutate(body, { onSuccess: () => void navigate("/admin/exams", { replace: true }) });
  }

  const textOf = (questionId: string) =>
    questions.find((question) => question.id === questionId)?.text ?? questionId;

  return (
    <>
      {/* The server checks the values and gives the messages, so the browser does not check them. */}
      <form className="mt-6 flex flex-col gap-4" onSubmit={submit} noValidate>
        <div>
          <label htmlFor="title">Title</label>
          <input
            id="title"
            className={fieldClass}
            type="text"
            value={form.title}
            onChange={(event) => setForm({ ...form, title: event.target.value })}
          />
        </div>
        <div>
          <label htmlFor="opensAt">Open time</label>
          <input
            id="opensAt"
            className={fieldClass}
            type="datetime-local"
            value={form.opensAt}
            onChange={(event) => setForm({ ...form, opensAt: event.target.value })}
          />
        </div>
        <div>
          <label htmlFor="closesAt">Close time</label>
          <input
            id="closesAt"
            className={fieldClass}
            type="datetime-local"
            value={form.closesAt}
            onChange={(event) => setForm({ ...form, closesAt: event.target.value })}
          />
        </div>
        <div>
          <label htmlFor="timeLimit">Time limit in minutes</label>
          <input
            id="timeLimit"
            className={fieldClass}
            type="number"
            min={1}
            max={480}
            value={form.timeLimit}
            onChange={(event) => setForm({ ...form, timeLimit: event.target.value })}
          />
        </div>
        <fieldset className="flex flex-col gap-2">
          <legend className="font-bold">Questions</legend>
          {questions.length === 0 && (
            <p className="text-gray-700">
              The Question bank has no Questions. Create a Question first.{" "}
              <Link className="underline" to="/admin/questions/new">
                New Question
              </Link>
            </p>
          )}
          {questions.map((question) => (
            <label key={question.id} className="flex items-start gap-2">
              <input
                type="checkbox"
                checked={form.questionIds.includes(question.id)}
                onChange={() => toggle(question.id)}
              />
              {question.text}
            </label>
          ))}
        </fieldset>
        {form.questionIds.length > 0 && (
          <div>
            <p className="font-bold">The Exam shows the Questions in this order.</p>
            <ol aria-label="Selected Questions" className="list-decimal pl-6">
              {form.questionIds.map((questionId) => (
                <li key={questionId}>{textOf(questionId)}</li>
              ))}
            </ol>
          </div>
        )}
        <div className="flex gap-3">
          {/* The button uses aria-disabled, not disabled: a disabled button loses the keyboard focus. submit ignores a second send. */}
          <button
            className="rounded bg-blue-800 px-4 py-2 text-white aria-disabled:opacity-50"
            type="submit"
            aria-disabled={save.isPending}
          >
            Save Exam
          </button>
          <Link className="rounded border border-gray-500 px-4 py-2" to="/admin/exams">
            Cancel
          </Link>
        </div>
      </form>
      {/* The region is always in the page. A screen reader announces the text that appears in a region that was there before. */}
      <div className="mt-4 text-red-800 empty:mt-0" role="alert">
        {save.isError && messageOf(save.error)}
      </div>
    </>
  );
}

// ExamLinkPanel shows the Exam link of a saved Exam. The Admin copies it and gives it to the Candidates.
function ExamLinkPanel({ exam }: { exam: Exam }) {
  const regenerate = useRegenerateExamLink(exam.id);
  const [confirming, setConfirming] = useState(false);
  const [copy, setCopy] = useState<"idle" | "copied" | "failed">("idle");
  const link = `${window.location.origin}/e/${exam.linkToken}`;

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(link);
      setCopy("copied");
    } catch {
      setCopy("failed");
    }
  }

  function confirmRegenerate() {
    if (regenerate.isPending) {
      return;
    }
    setCopy("idle");
    regenerate.mutate(undefined, { onSettled: () => setConfirming(false) });
  }

  return (
    <section aria-labelledby="exam-link-heading" className="mt-10 border-t border-gray-300 pt-6">
      <h2 id="exam-link-heading" className="text-xl font-bold">
        Exam link
      </h2>
      <p className="mt-2 break-all">
        <code>{link}</code>
      </p>
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <button className={buttonClass} type="button" onClick={() => void copyLink()}>
          Copy link
        </button>
        {confirming ? (
          <>
            <p>Regenerate the link? The old link stops working.</p>
            <button
              className={`${buttonClass} text-red-800`}
              type="button"
              aria-disabled={regenerate.isPending}
              onClick={confirmRegenerate}
            >
              Confirm regenerate
            </button>
            <button className={buttonClass} type="button" onClick={() => setConfirming(false)}>
              Cancel regenerate
            </button>
          </>
        ) : (
          <button className={buttonClass} type="button" onClick={() => setConfirming(true)}>
            Regenerate link
          </button>
        )}
        <Link className={buttonClass} to={`/admin/exams/${exam.id}/results`}>
          Results
        </Link>
      </div>
      <div className="mt-3 empty:mt-0" role="status">
        {copy === "copied" && "The link is copied."}
      </div>
      <div className="mt-3 text-red-800 empty:mt-0" role="alert">
        {copy === "failed" &&
          "The browser did not copy the link. Select the link and copy it by hand."}
        {regenerate.isError && messageOf(regenerate.error)}
      </div>
    </section>
  );
}
