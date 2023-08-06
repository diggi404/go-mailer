// Establish WebSocket connection when the page loads
const socket = new WebSocket("ws://localhost:3000/ws");

socket.onopen = (event) => {
  console.log("WebSocket connection established.");
};

socket.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log("Received update:", message);
};

function submitForm() {
  const form = document.getElementById("mail-form");
  const formData = new FormData(form);

  fetch("/", {
    method: "POST",
    body: formData,
  })
    .then((response) => {
      if (response.ok) {
        console.log(response);
      } else {
        console.error("Error submitting form.");
      }
    })
    .catch((error) => {
      console.error("An error occurred:", error);
    });
}

const form = document.getElementById("mail-form");
form.addEventListener("submit", function (event) {
  event.preventDefault();
  submitForm();
});
