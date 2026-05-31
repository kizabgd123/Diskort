from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path):
    return (ROOT / path).read_text(encoding="utf-8")


def test_contains(path, needle):
    text = read(path)
    assert needle in text, f"{needle!r} missing from {path}"


def main():
    test_contains("src/services/reservations.js", "reservationCollection = 'fkSava45Reservations'")
    test_contains("src/services/reservations.js", "documentId=${slot.id}")
    test_contains("src/services/reservations.js", "googleCalendarUrl")
    test_contains("src/services/analytics.js", "window.dataLayer")
    test_contains("src/services/analytics.js", "window.fbq")
    test_contains("src/main.js", "Book Online")
    test_contains("src/main.js", "Query latency")
    test_contains("src/styles.css", "@media (max-width: 860px)")
    test_contains("docs/fk-sava-45-system.md", "Stripe Billing")
    test_contains("docs/fk-sava-45-system.md", "Square")
    print("static checks passed")


if __name__ == "__main__":
    main()
