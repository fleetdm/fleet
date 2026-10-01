import React, { Suspense } from "react";

import Spinner from "components/Spinner";

import type { EditorMode, IEditorProps } from "./Editor";

// Shares the ace-editor chunk with the other ace-based editors — see
// components/SQLEditor/index.tsx.
const Editor = React.lazy(
  () => import(/* webpackChunkName: "ace-editor" */ "./Editor")
);

const LazyEditor = (props: IEditorProps): JSX.Element => (
  <Suspense fallback={<Spinner verticalPadding="small" />}>
    <Editor {...props} />
  </Suspense>
);

export type { EditorMode, IEditorProps };
export default LazyEditor;
