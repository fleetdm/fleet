import { createContext } from "react";

/** "small" renders the 12px external link icon, for links in 12px help text. */
const CustomLinkSizeContext = createContext<"default" | "small">("default");

export default CustomLinkSizeContext;
