import { createContext } from "react";

/** "small" renders the 12px external link icon, for links in 12px help text and tooltips. */
const CustomLinkSizeContext = createContext<"default" | "small">("default");

export default CustomLinkSizeContext;
