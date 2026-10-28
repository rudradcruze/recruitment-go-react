import { useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { messageOf } from "../../api/errors";
import { useHeadingFocus } from "../../components/use-heading-focus";
import { usePageTitle } from "../../components/use-page-title";
import { useQuestion, useSaveQuestion, type Question } from "./use-questions";

const minOptions = 2;
const maxOptions = 6;
const fieldClass = "mt-1 w-full rounded border border-gray-500 px-3 py-2";
const buttonClass = "rounded border border-gray-500 px-3 py-1 aria-disabled:opacity-50";

interface FormState {
  text: string;
  options: string[];
  // The index of the correct Option, or null when the person has not marked one. The API then explains the rule.
  correct: number | null;
}

function stateOf(question: Question | undefined): FormState {
  if (question === undefined) {
    return { text: "", options: ["", ""], correct: null };
  }
  const correct = question.options.findIndex((option) => option.correct);
  return {
    text: question.text,
    options: question.options.map((option) => option.text),
    correct: correct === -1 ? null : correct,
  };
}

// QuestionFormPage makes a new Question, or changes the Question that the address names.
export function QuestionFormPage() {
  const { id } = useParams();
  const question = useQuestion(id);
  usePageTitle(id === undefined ? "New Question" : "Edit Question");
  const heading = useHeadingFocus();

  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold focus:outline-none">
        {id === undefined ? "New Question" : "Edit Question"}
      </h1>
      {id !== undefined && question.data === undefined && (
        <>
          <div className="mt-4 empty:mt-0" role="status">
            {question.isPending && "The application loads the Question."}
          </div>
          <div className="mt-4 text-red-800 empty:mt-0" role="alert">
            {question.isError && messageOf(question.error)}
          </div>
        </>
      )}
      {(id === undefined || question.data !== undefined) && (
        <QuestionForm
          key={question.data?.id ?? "new"}
          questionId={id}
          initial={stateOf(question.data)}
        />
      )}
    </main>
  );
}

function QuestionForm({
  questionId,
  initial,
}: {
  questionId: string | undefined;
  initial: FormState;
}) {
  const [form, setForm] = useState<FormState>(initial);
  const save = useSaveQuestion(questionId);
  const navigate = useNavigate();

  function setOption(index: number, text: string) {
    setForm({ ...form, options: form.options.map((option, i) => (i === index ? text : option)) });
  }

  function addOption() {
    if (form.options.length < maxOptions) {
      setForm({ ...form, options: [...form.options, ""] });
    }
  }

  function removeOption(index: number) {
    const correct =
      form.correct === null || form.correct === index
        ? null
        : form.correct > index
          ? form.correct - 1
          : form.correct;
    setForm({ ...form, options: form.options.filter((_, i) => i !== index), correct });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (save.isPending) {
      return;
    }
    const body = {
      text: form.text,
      options: form.options.map((text, i) => ({ text, correct: i === form.correct })),
    };
    // Replace the page in the history, so the Back button does not return to the form.
    save.mutate(body, { onSuccess: () => void navigate("/admin/questions", { replace: true }) });
  }

  return (
    <>
      {/* The server checks the values and gives the messages, so the browser does not check them. */}
      <form className="mt-6 flex flex-col gap-4" onSubmit={submit} noValidate>
        <div>
          <label htmlFor="text">Question text</label>
          <textarea
            id="text"
            className={fieldClass}
            rows={3}
            value={form.text}
            onChange={(event) => setForm({ ...form, text: event.target.value })}
          />
        </div>
        <fieldset className="flex flex-col gap-3">
          <legend className="font-bold">Options</legend>
          {form.options.map((option, index) => (
            <div key={index} className="flex flex-wrap items-end gap-3">
              <div className="min-w-48 flex-1">
                <label htmlFor={`option-${index}`}>{`Option ${index + 1}`}</label>
                <input
                  id={`option-${index}`}
                  className={fieldClass}
                  type="text"
                  value={option}
                  onChange={(event) => setOption(index, event.target.value)}
                />
              </div>
              <label className="flex items-center gap-2 py-2">
                <input
                  type="radio"
                  name="correct"
                  aria-label={`Option ${index + 1} is correct`}
                  checked={form.correct === index}
                  onChange={() => setForm({ ...form, correct: index })}
                />
                Correct
              </label>
              {form.options.length > minOptions && (
                <button
                  className={buttonClass}
                  type="button"
                  aria-label={`Remove Option ${index + 1}`}
                  onClick={() => removeOption(index)}
                >
                  Remove
                </button>
              )}
            </div>
          ))}
          <div>
            <button
              className={buttonClass}
              type="button"
              aria-disabled={form.options.length >= maxOptions}
              onClick={addOption}
            >
              Add Option
            </button>
            {form.options.length >= maxOptions && (
              <span className="ml-3 text-gray-700">A Question has up to {maxOptions} Options.</span>
            )}
          </div>
        </fieldset>
        <div className="flex gap-3">
          {/* The button uses aria-disabled, not disabled: a disabled button loses the keyboard focus. submit ignores a second send. */}
          <button
            className="rounded bg-blue-800 px-4 py-2 text-white aria-disabled:opacity-50"
            type="submit"
            aria-disabled={save.isPending}
          >
            Save Question
          </button>
          <Link className="rounded border border-gray-500 px-4 py-2" to="/admin/questions">
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
