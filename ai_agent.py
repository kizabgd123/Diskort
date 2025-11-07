import google.generativeai as genai
import config
import json

class AIAgent:
    def __init__(self):
        if not config.GOOGLE_API_KEY:
            raise ValueError("Google AI API ključ nije pronađen.")
        genai.configure(api_key=config.GOOGLE_API_KEY)
        self.model = genai.GenerativeModel('models/gemini-2.5-flash')

    def analyze_message_for_relocation(self, content: str):
        """
        Koristi AI da proceni da li poruku treba premestiti.
        Vraća JSON sa odlukom i razlogom.
        """
        prompt = f"""
        Ti si AI moderator za Discord. Tvoj zadatak je da analiziraš poruku i odlučiš da li je treba premestiti u poseban kanal za važne objave.

        Pravila za odlučivanje:
        - Premesti poruku ako je to: blog post, članak, duža vest, detaljan vodič, ili link ka nečemu od toga sa opisom.
        - Premesti poruku ako ima jasan naslov i više od 30 reči.
        - IGNORIŠI kratke, neformalne poruke, pitanja, meme-ove ili proste pozdrave.

        Tvoj odgovor MORA biti u JSON formatu sa dva ključa:
        1. "premesti": boolean (true/false)
        2. "razlog": string (kratko objašnjenje odluke, npr. "Prepoznat kao članak.")

        Primer 1:
        Poruka: "Evo super tutorijala o Pythonu koji sam našao: [link]"
        Tvoj odgovor: {{"premesti": true, "razlog": "Link ka tutorijalu."}}

        Primer 2:
        Poruka: "lol dobar meme"
        Tvoj odgovor: {{"premesti": false, "razlog": "Neformalna poruka."}}

        Sada analiziraj sledeću poruku:
        Poruka: "{content}"
        """
        try:
            response = self.model.generate_content(prompt)
            # Čistimo odgovor da bismo dobili samo JSON
            cleaned_response = response.text.strip().replace("```json", "").replace("```", "")
            decision = json.loads(cleaned_response)
            return decision
        except Exception as e:
            print(f"AI analiza neuspešna: {e}")
            return {"premesti": False, "razlog": "Greška u AI analizi."}
