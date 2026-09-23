// Pyntra theme toggle: light / dark with OS-preference fallback.
(function () {
 var STORAGE_KEY = 'pyntra_theme';

 function stored() {
  try { return localStorage.getItem(STORAGE_KEY); } catch (e) { return null; }
 }

 function systemPrefersDark() {
  return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
 }

 // The effective theme = explicit override if set, otherwise the OS preference.
 function effectiveTheme() {
  var s = stored();
  if (s === 'dark' || s === 'light') return s;
  return 'dark'; // Pyntra defaults to the dark, enterprise look.
 }

 function apply(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  try { localStorage.setItem(STORAGE_KEY, theme); } catch (e) {}
 }

 window.toggleTheme = function () {
  apply(effectiveTheme() === 'dark' ? 'light' : 'dark');
 };

 // Keep in sync with OS changes only while the user hasn't chosen explicitly.
 if (window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
   if (!stored()) document.documentElement.removeAttribute('data-theme');
  });
 }
})();
