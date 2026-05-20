import { useCallback, useState } from 'react';

/** Discourage browser / extension password managers on admin forms. */
export const noAutofillAttrs = {
  'data-lpignore': 'true',
  'data-1p-ignore': true,
  'data-form-type': 'other',
};

export const noAutofillFormProps = {
  autoComplete: 'off',
};

/** Username, user ID, and other non-secret text on non-login forms. */
export const noAutofillTextProps = {
  autoComplete: 'off',
  ...noAutofillAttrs,
};

/** Password fields on admin create/edit forms. */
export const noAutofillPasswordProps = {
  autoComplete: 'new-password',
  ...noAutofillAttrs,
};

/** API keys, access tokens, and other secrets (often type="password"). */
export const noAutofillSecretProps = {
  autoComplete: 'one-time-code',
  ...noAutofillAttrs,
};

/** Semantic UI Dropdown and similar controls that are not credential fields. */
export const noAutofillDropdownProps = {
  autoComplete: 'new-password',
  ...noAutofillAttrs,
};

/**
 * Chrome may autofill despite autocomplete; readonly until first focus
 * suppresses the initial saved-password popup.
 */
export function useNoAutofillUnlock() {
  const [unlocked, setUnlocked] = useState(false);
  const onFocus = useCallback(() => setUnlocked(true), []);
  return { readOnly: !unlocked, onFocus };
}
