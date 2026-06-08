/** Discourage browser / extension password managers on admin forms. */
export const noAutofillAttrs = {
  'data-lpignore': 'true',
  'data-1p-ignore': true,
  'data-form-type': 'other',
};

export const noAutofillFormProps = {
  autocomplete: 'off',
};

/** Username, user ID, and other non-secret text on non-login forms. */
export const noAutofillTextProps = {
  autocomplete: 'off',
  ...noAutofillAttrs,
};

/** Password fields on admin create/edit forms. */
export const noAutofillPasswordProps = {
  autocomplete: 'new-password',
  ...noAutofillAttrs,
};

/** API keys, access tokens, and other secrets (often type="password"). */
export const noAutofillSecretProps = {
  autocomplete: 'one-time-code',
  ...noAutofillAttrs,
};

export const noAutofillDropdownProps = {
  autocomplete: 'new-password',
  ...noAutofillAttrs,
};

import { ref } from 'vue';

/**
 * Chrome may autofill despite autocomplete; keep the field readonly until first
 * focus to suppress the initial saved-password popup. Returns { readonly, onFocus }.
 */
export function useNoAutofillUnlock() {
  const unlocked = ref(false);
  const onFocus = () => {
    unlocked.value = true;
  };
  return { readonly: unlocked, onFocus };
}
