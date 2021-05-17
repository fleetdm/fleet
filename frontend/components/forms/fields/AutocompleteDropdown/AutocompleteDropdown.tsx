import React, { useCallback } from "react";
import { Async, OnChangeHandler, Option } from "react-select";
import classnames from "classnames";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import local from "utilities/local";
// @ts-ignore
import debounce from "utilities/debounce";

const baseClass = "autocomplete-dropdown";

interface IAutocompleteDropdown {
  id: string;
  placeholder: string;
  onChange: OnChangeHandler;
  resourceUrl: string;
  valueKey: string;
  labelKey: string;
  value: Option[];
  disabledOptions: number[];
  disabled?: boolean;
  className?: string;
}

const debounceOptions = {
  timeout: 300,
  leading: false,
  trailing: true,
};

const createUrl = (baseUrl: string, input: string) => {
  return `/api${baseUrl}?query=${input}`;
};

const AutocompleteDropdown = (props: IAutocompleteDropdown): JSX.Element => {
  const {
    className,
    disabled,
    disabledOptions,
    placeholder,
    onChange,
    id,
    resourceUrl,
    valueKey,
    labelKey,
    value,
  } = props;

  const wrapperClass = classnames(baseClass, className);

  // We disable any filtering client side as the server filters the results
  // for us.
  const filterOptions = useCallback((options) => {
    return options;
  }, []);

  // NOTE: It seems react-select v1 Async component does not work well with
  // returning results from promises in its loadOptions handler. That is why
  // we have decided to use callbacks as those seemed to make the component work
  // More info is here:
  // https://stackoverflow.com/questions/52984105/react-select-async-loadoptions-is-not-loading-options-properly
  const getOptions = debounce((input: string, callback: any) => {
    if (!input) {
      return callback([]);
    }

    fetch(createUrl(resourceUrl, input), {
      headers: {
        authorization: `Bearer ${local.getItem("auth_token")}`,
      },
    })
      .then((res) => {
        return res.json();
      })
      .then((json) => {
        // TODO: make more generic.
        const optionsData = json.users.map((user: any) => {
          if (disabledOptions.includes(user.id) || user.global_role !== null) {
            user.disabled = true;
          }
          return user;
        });
        callback(null, { options: optionsData });
      })
      .catch((err) => {
        console.log("There was an error", err);
      });
  }, debounceOptions);

  return (
    <div className={wrapperClass}>
      <Async
        noResultsText={"Nothing found"}
        autoload={false}
        cache={false}
        id={id}
        loadOptions={getOptions}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
        valueKey={valueKey}
        value={value}
        labelKey={labelKey}
        filterOptions={filterOptions}
        multi
        searchable
      />
    </div>
  );
};

export default AutocompleteDropdown;
