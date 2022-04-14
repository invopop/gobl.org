const favicon = document.getElementById("js-favicon");

function changeFavicon() {
  favicon.href = window.matchMedia("(prefers-color-scheme: dark)").matches 
    ? '/faviconLight.svg'
    : '/faviconDark.svg';
};

changeFavicon();

window
  .matchMedia('(prefers-color-scheme: dark)')
  .addEventListener('change', changeFavicon);