import React, { createContext, useCallback, useMemo, useState } from "react";

interface ITitleState {
  title?: string;
  locked: boolean;
}

interface ITitleContext extends ITitleState {
  setTitle: (title: string, options?: { lock?: boolean }) => void;
  clearTitle: () => void;
}

const defaultContext: ITitleContext = {
  title: undefined,
  locked: false,
  setTitle: () => undefined,
  clearTitle: () => undefined,
};

export const TitleContext = createContext<ITitleContext>(defaultContext);

interface ITitleProviderProps {
  children: React.ReactNode;
}

const TitleProvider = ({ children }: ITitleProviderProps): JSX.Element => {
  const [state, setState] = useState<ITitleState>({
    title: undefined,
    locked: false,
  });

  const setTitle = useCallback(
    (title: string, options?: { lock?: boolean }) => {
      setState({
        title,
        locked: !!options?.lock,
      });
    },
    []
  );

  const clearTitle = useCallback(() => {
    setState({
      title: undefined,
      locked: false,
    });
  }, []);

  const value = useMemo(
    () => ({
      ...state,
      setTitle,
      clearTitle,
    }),
    [state, setTitle, clearTitle]
  );

  return (
    <TitleContext.Provider value={value}>{children}</TitleContext.Provider>
  );
};

export default TitleProvider;
