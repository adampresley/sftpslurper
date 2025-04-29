import { Dialog } from "/static/js/dialog.min.js";

document.addEventListener('DOMContentLoaded', () => {
   attachPreviewClickListeners();

   document.body.addEventListener("htmx:afterSettle", () => {
      attachPreviewClickListeners();
   });
});

function attachPreviewClickListeners() {
   document.querySelectorAll(".fileLink").forEach(link => {
      link.addEventListener("click", async (e) => {
         const el = e.target;
         document.querySelector("#previewBody").innerHTML = await getPreviewContent(el);
         document.querySelector("#previewWindow").show();
      });
   });
}

async function getPreviewContent(el) {
   const root = el.dataset.root;
   const ext = el.dataset.ext;
   const name = el.dataset.name;

   const response = await fetch(`/preview?root=${root}&ext=${ext}&filename=${name}`);

   if (!response.ok) {
      alert(`Failed to load preview: ${response.status}`);
      return `<article class="error">Failed to load preview</article>`;
   }

   const result = await response.text();
   return result;
}

