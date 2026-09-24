import React, { Suspense } from "react";

import Spinner from "components/Spinner";

// Shares the ace-editor chunk with the other ace-based editors — see
// components/SQLEditor/index.tsx.
const YamlAce = React.lazy(() =>
  import(/* webpackChunkName: "ace-editor" */ "./YamlAce")
);

const LazyYamlAce = (props) => (
  <Suspense fallback={<Spinner verticalPadding="small" />}>
    <YamlAce {...props} />
  </Suspense>
);

export default LazyYamlAce;
