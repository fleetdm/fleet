import React, { useState } from "react";

import Button from "components/buttons/Button";

const baseClass = "generated-profile-summary";

// Match the name and description columns (server/datastore/mysql/schema.sql).
const NAME_MAX_LENGTH = 255;
const DESCRIPTION_MAX_LENGTH = 1023;

/** Shows `code` spans in the AI's description as chips. */
export const renderWithCodeChips = (text: string) =>
  text.split(/(`[^`]+`)/).map((part, i) =>
    part.length > 2 && part.startsWith("`") && part.endsWith("`") ? (
      // eslint-disable-next-line react/no-array-index-key
      <code key={i} className={`${baseClass}__chip`}>
        {part.slice(1, -1)}
      </code>
    ) : (
      part
    )
  );

interface IEditableTextProps {
  value: string;
  label: "Name" | "Description";
  placeholder: string;
  maxLength: number;
  multiline?: boolean;
  disabled?: boolean;
  onChange: (value: string) => void;
  renderValue: (value: string) => React.ReactNode;
}

/** Text that shows as a heading or paragraph and turns into a field when its
 * pencil is clicked. An empty value is shown as the field, so it can be
 * filled in. */
const EditableText = ({
  value,
  label,
  placeholder,
  maxLength,
  multiline,
  disabled,
  onChange,
  renderValue,
}: IEditableTextProps) => {
  const [draft, setDraft] = useState<string | null>(null);
  const isEditing = draft !== null || !value;

  const commit = () => {
    if (draft !== null) {
      onChange(draft.trim());
    }
    setDraft(null);
  };

  const onKeyDown = (
    e: React.KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    if (e.key === "Escape") {
      setDraft(null);
    } else if (e.key === "Enter" && !multiline) {
      e.preventDefault();
      commit();
    }
  };

  if (isEditing) {
    const fieldProps = {
      "aria-label": label,
      className: `${baseClass}__field`,
      value: draft ?? value,
      placeholder,
      maxLength,
      disabled,
      // a field shown because the value is empty waits for the admin
      autoFocus: draft !== null,
      onChange: (
        e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
      ) => setDraft(e.target.value),
      onBlur: commit,
      onKeyDown,
    };
    return multiline ? (
      <textarea {...fieldProps} rows={2} />
    ) : (
      <input {...fieldProps} type="text" />
    );
  }

  return (
    <div className={`${baseClass}__row`}>
      {renderValue(value)}
      <Button
        variant="subdued"
        icon="pencil"
        ariaLabel={`Edit ${label.toLowerCase()}`}
        disabled={disabled}
        onClick={() => setDraft(value)}
      />
    </div>
  );
};

interface IGeneratedProfileSummaryProps {
  name: string;
  description: string;
  /** Shows a placeholder bar until the value arrives. */
  isNamePending?: boolean;
  isDescriptionPending?: boolean;
  disabled?: boolean;
  onChangeName: (name: string) => void;
  onChangeDescription: (description: string) => void;
}

/** The AI-generated profile's title and summary, each editable in place. */
const GeneratedProfileSummary = ({
  name,
  description,
  isNamePending,
  isDescriptionPending,
  disabled,
  onChangeName,
  onChangeDescription,
}: IGeneratedProfileSummaryProps) => (
  <div className={baseClass}>
    {isNamePending ? (
      <div
        className={`${baseClass}__placeholder ${baseClass}__placeholder--title`}
        data-testid="name-placeholder"
        // the progress step announces what's loading
        aria-hidden
      />
    ) : (
      <EditableText
        value={name}
        label="Name"
        placeholder="E.g., Disable camera access for Firefox"
        maxLength={NAME_MAX_LENGTH}
        disabled={disabled}
        onChange={onChangeName}
        renderValue={(value) => (
          <h1 className={`${baseClass}__name`}>{value}</h1>
        )}
      />
    )}
    {isDescriptionPending ? (
      <div
        className={`${baseClass}__placeholder-lines`}
        data-testid="description-placeholder"
        aria-hidden
      >
        <div className={`${baseClass}__placeholder`} />
        <div
          className={`${baseClass}__placeholder ${baseClass}__placeholder--short`}
        />
      </div>
    ) : (
      <EditableText
        value={description}
        label="Description"
        placeholder="E.g., Blocks Firefox from accessing the camera on managed Macs, without affecting other apps."
        maxLength={DESCRIPTION_MAX_LENGTH}
        multiline
        disabled={disabled}
        onChange={onChangeDescription}
        renderValue={(value) => (
          <p className={`${baseClass}__description`}>
            {renderWithCodeChips(value)}
          </p>
        )}
      />
    )}
  </div>
);

export default GeneratedProfileSummary;
