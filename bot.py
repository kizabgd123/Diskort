import discord
from discord.ext import commands
import config
from ai_agent import AIAgent

KANAL_ZA_ANALIZU = "💬-general"
KANAL_ZA_BLOGOVE = "📰-blog-postovi"

intents = discord.Intents.default()
intents.message_content = True
intents.members = True

bot = commands.Bot(command_prefix='!', intents=intents)

try:
    ai_agent = AIAgent()
except ValueError as e:
    print(f"UPOZORENJE: {e}")
    ai_agent = None

@bot.event
async def on_ready():
    print(f'Moderator Bot je ulogovan kao {bot.user}')

@bot.event
async def on_message(message):
    if message.author == bot.user:
        return
    await bot.process_commands(message)

    if ai_agent and message.channel.name == KANAL_ZA_ANALIZU:
        print(f"\n[DEBUG] Primljena poruka u kanalu '{KANAL_ZA_ANALIZU}'. Sadržaj: '{message.content[:50]}...'")
        print("[DEBUG] Šaljem na AI analizu...")
        analysis = ai_agent.analyze_message_for_relocation(message.content)
        print(f"[DEBUG] Dobijen AI odgovor: {analysis}")

        if analysis.get("premesti"):
            print(f"[DEBUG] AI Odluka: Premesti. Razlog: {analysis.get('razlog')}")
            destination_channel = discord.utils.get(message.guild.text_channels, name=KANAL_ZA_BLOGOVE)
            if not destination_channel:
                print(f"Greška: Kanal '{KANAL_ZA_BLOGOVE}' nije pronađen.")
                return
            embed = discord.Embed(title="📰 Premešten Sadržaj", description=message.content, color=discord.Color.blue())
            embed.set_author(name=message.author.display_name, icon_url=message.author.avatar.url)
            embed.set_footer(text=f"Originalno postavljeno u #{message.channel.name} | Razlog: {analysis.get('razlog')}")
            await destination_channel.send(embed=embed)
            await message.delete()
            await message.channel.send(f"Hej {message.author.mention}, tvoj post je premešten u {destination_channel.mention} da bi bio vidljiviji! ✨", delete_after=15)
        else:
            print(f"[DEBUG] AI Odluka: Ne premeštaj. Razlog: {analysis.get('razlog')}")
            await message.channel.send(f"🤖 *AI je analizirao poruku i odlučio da je ne premesti. (Razlog: {analysis.get('razlog')})*", delete_after=10)

@bot.command(name='setup')
@commands.has_permissions(administrator=True)
async def setup(ctx):
    """(FINALNA VERZIJA) Koristi discord.utils.get za pouzdanu proveru."""
    guild = ctx.guild
    await ctx.send("Proveravam i postavljam strukturu servera (Finalna Verzija)... ⏳")

    structure = {
        "📜 INFORMACIJE": {"kanali": ["dobrodošlica", "pravila", "objave"]},
        "💬 ZAJEDNICA": {"kanali": ["💬-general", "🎨-kreativni-kutak", "🤖-bot-komande"]},
        "💡 SADRŽAJ": {"kanali": ["📰-blog-postovi", "🔗-korisni-linkovi"]},
        "🔊 GLASOVNI KANALI": {"voice": ["General", "Gaming"]}
    }

    for category_name, content in structure.items():
        category_obj = discord.utils.get(guild.categories, name=category_name)
        if category_obj is None:
            category_obj = await guild.create_category(category_name)
            await ctx.send(f"Kreirana kategorija: `{category_name}`")
        
        if "kanali" in content:
            for channel_name in content["kanali"]:
                if discord.utils.get(guild.text_channels, name=channel_name) is None:
                    await guild.create_text_channel(channel_name, category=category_obj)
                    await ctx.send(f"  - Kreiran tekstualni kanal: `#{channel_name}`")
        
        if "voice" in content:
            for channel_name in content["voice"]:
                if discord.utils.get(guild.voice_channels, name=channel_name) is None:
                    await guild.create_voice_channel(channel_name, category=category_obj)
                    await ctx.send(f"  - Kreiran glasovni kanal: `{channel_name}`")

    await ctx.send("✅ Provera i postavljanje servera je završeno!")

@setup.error
async def setup_error(ctx, error):
    if isinstance(error, commands.MissingPermissions):
        await ctx.send("Greška: Nemaš administratorske dozvole za korišćenje ove komande.")

def run_bot():
    if not config.DISCORD_TOKEN: print("Greška: TOKEN nije podešen!"); return
    if not ai_agent: print("UPOZORENJE: Google AI Agent nije pokrenut.");
    print("Pokretanje Moderator Bota...")
    bot.run(config.DISCORD_TOKEN)

if __name__ == '__main__':
    run_bot()
