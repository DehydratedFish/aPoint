let date = new Date();
let selected_event = null;
let selected_day   = 0;

let days;

document.getElementById('previous_month').addEventListener('click', () => {
    date.setMonth(date.getMonth() - 1);
    render_calendar();
});

document.getElementById('next_month').addEventListener('click', () => {
    date.setMonth(date.getMonth() + 1);
    render_calendar();
});

function start_at_monday(day) {
    return !day ? 6 : day - 1;
}

async function get_events_current_month() {
    const date_string = `${date.getFullYear()}-${pad_zero(date.getMonth() + 1)}`;

    let response = await fetch(`/events/${date_string}`);
    let events = await response.json();

    if (events) {
        for (const event of events) {
            const day = Number(event.date.substr(-2))

            days[day].events.push(event)
        }
    }

    update_events(selected_day);
}

function reset_days() {
    days = new Array(32);

    for (let i = 1; i < 32; i += 1) {
        days[i] = {};
        days[i].events = new Array();
    }
}

function render_calendar() {
    reset_days();

    date.setDate(1);

    const body_div  = document.getElementById('calendar_body');
    const month_div = document.getElementById('month');

    const last_day_month = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
    const last_day_previous_month = new Date(date.getFullYear(), date.getMonth(), 0).getDate();

    const first_day = start_at_monday(date.getDay());
    const last_day  = start_at_monday(new Date(date.getFullYear(), date.getMonth() + 1, 0).getDay());

    const next_days = 7 - last_day - 1;

    const current_date = new Date();
    if (current_date.getMonth() === date.getMonth() && current_date.getFullYear() === date.getFullYear()) {
        selected_day = current_date.getDate();
    } else {
        selected_day = 0;
    }

    const months = [
        'Januar',
        'Februar',
        'März',
        'April',
        'Mai',
        'Juni',
        'Juli',
        'August',
        'September',
        'Oktober',
        'November',
        'Dezember',
    ];

    month_div.innerText = `${months[date.getMonth()]} ${date.getFullYear()}`;

    body_div.innerHTML = '';

    for (let i = first_day; i > 0; i -= 1) {
        const element = document.createElement('div');
        element.classList.add('previous_month_day');
        element.innerHTML = `${last_day_previous_month - i + 1}`;

        body_div.appendChild(element);
    }

    for (let i = 1; i <= last_day_month; i += 1) {
        const element = document.createElement('div');
        element.innerHTML = `${i}`;
        element.addEventListener('click', () => {
            update_events(i);
        });

        if (i === new Date().getDate() && date.getMonth() === new Date().getMonth()) {
            element.classList.add('today');
        }

        if (i === selected_day) {
            element.classList.add('selected_day');
        }

        days[i].div = element;
        body_div.appendChild(element);
    }

    for (let i = 1; i <= next_days; i += 1) {
        const element = document.createElement('div');
        element.classList.add('next_month_day');
        element.innerHTML = `${i}`;

        body_div.appendChild(element);
    }

    get_events_current_month();
}

let calendar_main = document.getElementById('calendar');
let event_popup   = document.getElementById('event_popup');

function update_events(day) {
    const events_div = document.getElementById('day_events');
    events_div.innerHTML = '';

    if (day === 0) {
        return;
    } else {
        if (selected_day) days[selected_day].div.classList.remove('selected_day');
        selected_day = day;
        days[selected_day].div.classList.add('selected_day');
    }

    const add_button = document.createElement('button');
    add_button.innerText = '+';
    add_button.setAttribute('id', 'add_event_button');
    add_button.addEventListener('click', () => {
        open_event_popup(null);
    });
    events_div.appendChild(add_button);

    for (const event of days[selected_day].events) {
        const element = document.createElement('div');
        element.innerHTML = `${event.name}\n${event.begin} - ${event.end}`;
        element.addEventListener('click', () => {
            open_event_popup(event);
        });

        events_div.appendChild(element);
    }
}

function open_event_popup(event) {
    calendar_main.style.display  = 'none';
    event_popup.style.visibility = 'visible';

    if (event) {
        const form = document.forms['event_form'];

        form.id.value     = event.id;
        form.name.value   = event.name;
        form.phone.value  = event.phone;
        form.email.value  = event.email;
        form.title.value  = event.title;
        form.notes.value  = event.notes;
        form.begin.value  = event.begin;
        form.end.value    = event.end;
        form.notify.checked = event.notify;

        notification_status = document.getElementById('notification_status');
        if (event.send) {
            notification_status.innerText = `Erinnerung gesendet ${event.send}.`;
        } else {
            if (event.notify) {
                notification_status.innerText = 'Terminerinnerung wird bald versendet.';
            } else {
                notification_status.innerText = '';
            }
        }
    }

    selected_event = event;
}

function close_event_popup() {
    calendar_main.style.display  = 'block';
    event_popup.style.visibility = 'hidden';

    selected_event = null;
    document.forms['event_form'].reset();

}

function pad_zero(number) {
    if (number < 10) return `0${number}`;

    return number.toString();
}

async function save_event(submit_event) {
    submit_event.preventDefault();

    const form = document.forms['event_form'];

    if (form.checkValidity()) {
        if (!selected_event) {
            selected_event = {};
        }

        selected_event.name   = form.name.value;
        selected_event.date   = `${date.getFullYear()}-${pad_zero(date.getMonth() + 1)}-${pad_zero(selected_day)}`;
        selected_event.phone  = form.phone.value;
        selected_event.email  = form.email.value;
        selected_event.title  = form.title.value;
        selected_event.notes  = form.notes.value;
        selected_event.begin  = form.begin.value;
        selected_event.end    = form.end.value;

        selected_event.notify = form.notify.checked;

        // TODO: error handling
        if (selected_event.id) {
            await fetch('/event', {method: 'PUT', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(selected_event)});
        } else {
            let response = await fetch('/event', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(selected_event)});
            selected_event.id = Number(await response.text());

            days[selected_day].events.push(selected_event)
        }

        form.reset();

        close_event_popup();
        update_events(selected_day);
    }
}

render_calendar();

