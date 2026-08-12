import React, { useCallback, useState } from "react";
import { Ace } from "ace-builds";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Editor from "components/Editor";
import Icon from "components/Icon";

const baseClass = "profile-advanced-options";

interface IProfileAdvancedOptionsProps {
  /** the custom activation JSON currently in the editor. Fleet synthesizes a
   * simple activation when this isn't supplied. */
  customActivation: string;
  onChangeCustomActivation: (value: string) => void;
  /** example activation shown while the editor is empty. A placeholder rather
   * than a seeded value so an admin who doesn't write one submits nothing and
   * Fleet synthesizes its default. */
  placeholder?: string;
  /** validation error to render in the editor's label slot. */
  error?: string | null;
  onBlurCustomActivation?: () => void;
  /** makes the activation read-only (e.g. GitOps mode). The section can still
   * be expanded so the current activation stays readable. */
  readOnly?: boolean;
}

/** Advanced options for an Apple DDM (declaration) upload. Renders a custom
 * activation the admin can edit to control which hosts the declaration applies
 * to. Callers are responsible for showing this only for DDM uploads on
 * Premium. */
const ProfileAdvancedOptions = ({
  customActivation,
  onChangeCustomActivation,
  placeholder,
  error,
  onBlurCustomActivation,
  readOnly,
}: IProfileAdvancedOptionsProps) => {
  const [isShowing, setIsShowing] = useState(false);

  // the activation is a short snippet rather than a file being navigated, so
  // the line number gutter and indent guides are noise.
  const onEditorLoad = useCallback((editor: Ace.Editor) => {
    editor.renderer.setShowGutter(false);
    editor.setOption("displayIndentGuides", false);
  }, []);

  return (
    <div className={baseClass}>
      <Button
        className={`${baseClass}__toggle`}
        variant="subdued"
        ariaExpanded={isShowing}
        onClick={() => setIsShowing(!isShowing)}
      >
        Advanced options
        <Icon
          name={isShowing ? "chevron-up" : "chevron-down"}
          color="core-fleet-black"
        />
      </Button>
      {isShowing && (
        // no `mode` is set: ace registers syntax modes globally, so pulling in
        // mode-json here would also change how JSON renders in other editors
        // (e.g. the software configuration modal).
        <Editor
          className={`${baseClass}__custom-activation`}
          name="custom-activation"
          label="Custom activation"
          value={customActivation}
          placeholder={placeholder}
          // the example placeholder is 9 lines, and an empty editor would
          // otherwise collapse to 2 and clip it.
          minLines={9}
          error={error}
          readOnly={readOnly}
          onChange={onChangeCustomActivation}
          onBlur={onBlurCustomActivation}
          onLoad={onEditorLoad}
          helpText={
            <>
              Fleet creates a simple activation by default. Provide your own for
              advanced setups.{" "}
              <CustomLink
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/ddm-activations`}
                text="Learn more"
                newTab
              />
            </>
          }
        />
      )}
    </div>
  );
};

export default ProfileAdvancedOptions;
