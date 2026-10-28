import type { RouteObject } from "react-router";

import { AdminHomePage } from "../features/admin/admin-home-page";
import { AdminLayout } from "../features/admin/admin-layout";
import { LoginPage } from "../features/admin/login-page";
import { AttemptPage } from "../features/exam/attempt-page";
import { ExamFormPage } from "../features/exam/exam-form-page";
import { ExamLinkPage } from "../features/exam/exam-link-page";
import { ExamListPage } from "../features/exam/exam-list-page";
import { ExamResultsPage } from "../features/exam/exam-results-page";
import { QuestionFormPage } from "../features/exam/question-form-page";
import { QuestionListPage } from "../features/exam/question-list-page";
import { HomePage } from "./home-page";

export const routes: RouteObject[] = [
  { path: "/", element: <HomePage /> },
  { path: "/e/:token", element: <ExamLinkPage /> },
  { path: "/attempts/:id", element: <AttemptPage /> },
  { path: "/admin/login", element: <LoginPage /> },
  {
    path: "/admin",
    element: <AdminLayout />,
    children: [
      { index: true, element: <AdminHomePage /> },
      { path: "questions", element: <QuestionListPage /> },
      { path: "questions/new", element: <QuestionFormPage /> },
      { path: "questions/:id", element: <QuestionFormPage /> },
      { path: "exams", element: <ExamListPage /> },
      { path: "exams/new", element: <ExamFormPage /> },
      { path: "exams/:id", element: <ExamFormPage /> },
      { path: "exams/:id/results", element: <ExamResultsPage /> },
    ],
  },
];
