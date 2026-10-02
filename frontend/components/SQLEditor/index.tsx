import React, { Suspense } from "react";

import Spinner from "components/Spinner";

import type { ISQLEditorProps } from "./SQLEditor";

/**
 * ace-builds and react-ace are ~2.7 MB of modules. Loading them here rather
 * than at the import site defers them past route entry to the point an editor
 * actually renders — most consumers are modals and forms that are never opened,
 * and it keeps ace out of every route chunk that merely might show an editor.
 */
const SQLEditor = React.lazy(
  () => import(/* webpackChunkName: "ace-editor" */ "./SQLEditor")
);

const LazySQLEditor = (props: ISQLEditorProps): JSX.Element => (
  <Suspense fallback={<Spinner verticalPadding="small" />}>
    <SQLEditor {...props} />
  </Suspense>
);

export type { ISQLEditorProps };
export default LazySQLEditor;
