const form = document.querySelector(".contact-form")
const formStatus = document.querySelector(".form-status")


form.addEventListener("submit", async (event) => {
	event.preventDefault();
	formStatus.textContent = "Sending your message...";
	try {
		const formData = new FormData(form)
		const response = await fetch(form.action, {
			method: form.method,
			body: formData,
		});
		const result = await response.json();
		if (result.success) {
			formStatus.textContent = "Thanks for reaching out! We'll be in touch soon.";
			form.reset();
		} else {

			formStatus.textContent = "Something went wrong. Please try again or email us directly at itqanwebstudio@gmail.com";
		}

	} catch (error) {

		formStatus.textContent = "Something went wrong. Please try again or email us directly at itqanwebstudio@gmail.com";
	}
});
