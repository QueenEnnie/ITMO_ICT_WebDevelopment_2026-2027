const issued = document.querySelector('input[name="issued"]');
const due = document.querySelector('input[name="due"]');

function validateDates() {
    due.min = issued.value;
    due.setCustomValidity(
        issued.value && due.value && due.value < issued.value
            ? 'Срок сдачи не может быть раньше даты выдачи.'
            : ''
    );
}

issued.addEventListener('input', validateDates);
issued.addEventListener('change', validateDates);
due.addEventListener('input', validateDates);
due.addEventListener('change', validateDates);
validateDates();
