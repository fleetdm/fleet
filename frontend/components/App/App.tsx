import React from "react";
import { useDispatch, useSelector } from "react-redux";
import classnames from "classnames";
import TableProvider from "context/table";

import { QueryClient, QueryClientProvider } from "react-query";

// @ts-ignore
import { authToken } from "utilities/local"; // @ts-ignore
import { useDeepEffect } from "utilities/hooks"; // @ts-ignore
import { fetchCurrentUser } from "redux/nodes/auth/actions"; // @ts-ignore
import { getConfig, getEnrollSecret } from "redux/nodes/app/actions";
import { IUser } from "interfaces/user";

interface IAppProps {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const App = ({ children }: IAppProps) => {
  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);
  const queryClient = new QueryClient();

  useDeepEffect(() => {
    if (!user && authToken()) {
      dispatch(fetchCurrentUser()).catch(() => false);
    }

    if (user) {
      dispatch(getConfig()).catch(() => false);
      dispatch(getEnrollSecret()).catch(() => false);
    }
  }, [user]);

  const wrapperStyles = classnames("wrapper");
  return (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <div className={wrapperStyles}>{children}</div>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;
