import { Dialog } from "/static/js/dialog.min.js";
import { Confirmer } from "/static/js/confirm.min.js";
import { Alerter } from "/static/js/alert.min.js";

document.addEventListener('DOMContentLoaded', () => {
   attachDeleteClickListeners();
   attachPreviewClickListeners();

   document.body.addEventListener("htmx:afterSettle", () => {
      attachDeleteClickListeners();
      attachPreviewClickListeners();
   });
});

function attachDeleteClickListeners() {
   document.querySelectorAll(".deleteLink").forEach(link => {
      link.addEventListener("click", async (e) => {
         const confirmer = new Confirmer();
         const alerter = new Alerter({ duration: 940000 });

         const el = e.target.parentElement;
         const root = el.dataset.root;
         const name = el.dataset.name;
         const isDir = el.dataset.isdir === "true";

         const choice = await confirmer.yesNo(`Are you sure you want to delete ${name}?`);

         console.log(choice);

         if (!choice) {
            return;
         }

         const options = {
            method: "DELETE",
         };

         const params = new URLSearchParams();
         params.append("root", root);
         params.append("name", name);
         params.append("isdir", isDir);

         const response = await fetch(`/uploads?${params}`, options);
         const result = await response.text();

         if (!response.ok) {
            alerter.error(`Failed to delete file: ${result}`);
            return;
         }

         window.location.reload();
      });
   });
}

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

