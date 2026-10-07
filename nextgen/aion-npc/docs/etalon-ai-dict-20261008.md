# 📚 ЭТАЛОННЫЙ СЛОВАРЬ AI-типов ↔ npc_templates.xml (R0, 08.10.2026)

> Артефакт R0 aion-npc. Сгенерирован скриптом [tools/extract_ai_dict.py](../tools/extract_ai_dict.py) по 4 Java-клонам `../../../../../reference/` (вне гита). Полные данные: [ai-dict-all-20261008.json](ai-dict-all-20261008.json). Воспроизвести (из nextgen/aion-npc): `python3 tools/extract_ai_dict.py > /tmp/ai_dict.json`.

## 1. Сводка по версиям

| Эталон | AI-handler'ов (`@AIName`) | NPC-шаблонов | AI-типов в XML | XML без handler'а | handler'ов вне XML |
|---|---|---|---|---|---|
| 7.7-Mobius | 143 | 106615 | **111** | 0 ✅ | 32 |
| 7.8-AL | 143 | 106669 | 111 | 0 | 32 |
| 7.8-AG | 143 | 106669 | 111 | 0 | 32 |
| 5.8-AG | 650 | 87761 | 568 | 0 | 82 |
| 4.8-BA | 457 | 63287 | 432 | 0 | 25 |

📌 **Выводы**: (1) AL 7.8 ≡ AG 7.8 в AI-слое (идентичные реестры и частоты) — достаточно ОДНОГО эталона 7.8. (2) Mobius 7.7 КОНСОЛИДИРОВАЛ AI: 568 типов в 5.8 → 111 в 7.7 (индивидуальные per-NPC скрипты свёрнуты в generic + classNpc/). (3) Везде 0 «XML без handler'а» — словарь ПОЛНЫЙ, «Bad AI names» не бывает. (4) `aggressive` (53К) + `general` (40К) = 88% NPC; хвост — спец-поведение (trap/portal/servant/siege...).

## 2. Механика (канон для мимикрии)

- **Регистрация**: AI-классы с `@AIName("...")` живут в `data/scripts/system/handlers/ai/*.java` (+ core `ai2/`: `NpcAI2`=@AIName("npc"), `AISummon`=@AIName("summon")); грузятся ScriptManager'ом по дескриптору `data/scripts/system/aihandlers.xml`.
- **Данные**: `npc_templates.xml` (в `static_data/npcs/`), атрибут `ai="..."` на `<npc_template>`; spawn'ы — `static_data/spawns/{Npcs,Instances,Bases,...}/<mapid>_<name>.xml` (spot x/y/z/h, respawn_time; walker_id/walker_index → `SpawnTemplate.walkerId/walkerIdx`).
- **Think**: НЕ в данных — event-driven: `AttackManager.performAttack` → SIMPLE/SKILL_ATTACK цикл по adelay/arate → `FINISH_ATTACK` → `npcAI.think()`; thinkLock (ReentrantLock) на AI-инстансе. Периодика в коде, не в XML ⇒ наш abyss-цикл 60с ищем дизasm-ом (R2).
- **Abyss**: класс `AbyssGuardSimpleAI2` есть в 7.7, но в XML НЕ используется (движковый/заменён); в 4.8 `simple_abyssguard` = 859 NPC (штатный тип). Abyss-охрана в live NC — в NPCSvr (теория ⏳).

## 3. Полный реестр 7.7/7.8 (111 AI-типов): ai → класс → частота (7.7 | 5.8 | 4.8)

| ai= | класс (Mobius 7.7) | 7.7 | 5.8 | 4.8 |
|---|---|---|---|---|
| `aggressive` | `AggressiveNpcAI2` | 53132 | 53039 | 39358 |
| `general` | `GeneralNpcAI2` | 40474 | 22238 | 11770 |
| `noaction` | `NoActionAI2` | 1930 | 2047 | 1695 |
| `flag` | `FlagAI2` | 1750 | 0 | 1566 |
| `artifact_protector` | `ArtifactProtectorAI2` | 1369 | 1174 | 966 |
| `dummy` | `DummyAI2` | 751 | 829 | 0 |
| `quest_use_item` | `QuestItemNpcAI2` | 705 | 696 | 610 |
| `useitem` | `ActionItemNpcAI2` | 515 | 521 | 489 |
| `portal` | `PortalAI2` | 440 | 434 | 379 |
| `trap` | `TrapNpcAI2` | 402 | 402 | 372 |
| `siege_protector` | `SiegeProtectorNpcAI2` | 379 | 367 | 0 |
| `portal_dialog` | `PortalDialogAI2` | 351 | 331 | 230 |
| `servant` | `ServantNpcAI2` | 337 | 340 | 301 |
| `artifact` | `ArtifactAI2` | 336 | 259 | 222 |
| `chest` | `ChestAI2` | 271 | 279 | 201 |
| `conquest_npc` | `Conquest_NpcAI2` | 224 | 224 | 0 |
| `fortressgate` | `SiegeFortressGateAI2` | 184 | 184 | 225 |
| `resurrect` | `ResurrectAI2` | 139 | 139 | 121 |
| `homing` | `HomingNpcAI2` | 120 | 120 | 121 |
| `siege_mine` | `MineAI2` | 99 | 99 | 99 |
| `book` | `BookAI2` | 85 | 85 | 85 |
| `groupgate` | `GroupGateAI2` | 81 | 81 | 80 |
| `onedmgperhit` | `OneDmgPerHitAI2` | 78 | 83 | 0 |
| `siege_shieldnpc` | `ShieldNpcAI2` | 78 | 78 | 78 |
| `skillarea` | `SkillAreaNpcAI2` | 65 | 65 | 65 |
| `kisk` | `KiskAI2` | 62 | 62 | 74 |
| `xdrakanpriest` | `DrakanPriestAI2` | 61 | 94 | 94 |
| `drakanmedic` | `DrakanMedicAI2` | 50 | 79 | 79 |
| `siege_gaterepair` | `GateRepairAI2` | 43 | 43 | 51 |
| `summoner` | `SummonerAI2` | 29 | 36 | 46 |
| `defensive_cannon` | `DefensiveCannonAI2` | 28 | 28 | 2 |
| `following` | `FollowingNpcAI2` | 27 | 27 | 26 |
| `spring` | `SpringAI2` | 27 | 27 | 30 |
| `siege_raceprotector` | `SiegeRaceProtectorAI2` | 25 | 25 | 19 |
| `fun_ride` | `FunRideAI2` | 24 | 24 | 0 |
| `dancer` | `DancerAI2` | 23 | 23 | 0 |
| `one_dmg` | `OneDmgAI2` | 21 | 21 | 0 |
| `quest_start_use_item` | `QuestStartItemNpcAi2` | 13 | 13 | 23 |
| `aggrorunner` | `WalkAggroRunnerAI2` | 12 | 12 | 0 |
| `instancetimer` | `InstanceTimerAI2` | 12 | 12 | 0 |
| `butler` | `ButlerAI2` | 12 | 12 | 12 |
| `altar_protector` | `AltarProtectorAI2` | 11 | 0 | 0 |
| `siege_weapon` | `SiegeWeaponAI2` | 8 | 8 | 6 |
| `agrint` | `AgrintAI2` | 8 | 8 | 8 |
| `postbox` | `PostboxAI2` | 8 | 8 | 7 |
| `world_blesser` | `WorldBlesserAI2` | 8 | 8 | 8 |
| `code_red_nurse` | `CodeRedNurserAI2` | 8 | 8 | 8 |
| `infiltrator` | `InfiltratorsAI2` | 7 | 7 | 7 |
| `invisiblekisk` | `InvisiblekiskAI2` | 7 | 7 | 0 |
| `portal_request` | `PortalRequestAI2` | 6 | 6 | 6 |
| `studioportal` | `StudioPortalAI2` | 6 | 6 | 6 |
| `housesign` | `HouseSignAI2` | 6 | 6 | 6 |
| `ice_sculptures` | `IceSculptureAI2` | 6 | 0 | 0 |
| `speaker` | `SpeakerAI2` | 5 | 5 | 5 |
| `bomb` | `BombAi2` | 5 | 5 | 7 |
| `ascensationquestnpc` | `AscensationNpcAI2` | 4 | 4 | 4 |
| `quest14026` | `Quest14026` | 4 | 4 | 0 |
| `helpers_agrint` | `HelpersAgrintAI2` | 4 | 4 | 0 |
| `dredgionCommander` | `DredgionCommanderAI2` | 4 | 4 | 0 |
| `incarnate` | `IncarnateAI2` | 4 | 4 | 0 |
| `haramelchest` | `HaramelChestAI2` | 4 | 4 | 0 |
| `krbuff` | `KromedesBuffAI2` | 4 | 4 | 4 |
| `snakecolors` | `SnakeColorsAI2` | 4 | 4 | 0 |
| `conquest_portal` | `Conquest_PortalAI2` | 4 | 4 | 0 |
| `homeward_bound_event` | `HomewardBoundEventAI2` | 4 | 4 | 0 |
| `conquest_buff` | `Conquest_BuffAI2` | 4 | 4 | 0 |
| `krobject` | `KromedesItemNpcsAI2` | 3 | 3 | 3 |
| `polorserin` | `PolorSerinAI2` | 2 | 2 | 2 |
| `aggressive_first_skill` | `AggressiveFirstSkillAI2` | 2 | 20 | 0 |
| `sacred_image` | `SacredImageAI2` | 2 | 2 | 0 |
| `enemyservant` | `EnemyServantAI2` | 2 | 2 | 2 |
| `kinquid_debuff` | `KinquidDebuffAI2` | 2 | 2 | 2 |
| `invisible_npc` | `InvisibleNpcAI2` | 2 | 2 | 0 |
| `krprisoners` | `KromedesPrisonersAI2` | 2 | 2 | 2 |
| `portal_elevator` | `PortalElevatorAI2` | 2 | 2 | 1 |
| `writhingcocoon` | `WrithingCocoonAI2` | 2 | 2 | 2 |
| `firecracker` | `FirecrackerAI2` | 2 | 2 | 2 |
| `housegate` | `HouseGateAI2` | 2 | 2 | 2 |
| `deliveryman` | `DeliveryManAI2` | 2 | 2 | 2 |
| `tallocssummon` | `TallocsSummonAI2` | 2 | 2 | 2 |
| `friendportal` | `FriendPortalAI2` | 2 | 2 | 2 |
| `buffer` | `BufferEventAI2` | 2 | 2 | 0 |
| `daeva_day_new` | `DaevaDayAI2` | 2 | 2 | 0 |
| `holytowerteleport` | `HolyTowerTeleportAI2` | 2 | 0 | 0 |
| `examscarecrow` | `ExamScareCrowAI2` | 2 | 0 | 0 |
| `naia` | `NaiaAI2` | 1 | 1 | 1 |
| `general_first_skill` | `GeneralFirstSkillAI2` | 1 | 1 | 0 |
| `generalrunner` | `WalkGeneralRunnerAI2` | 1 | 1 | 0 |
| `gale_cyclone` | `GaleCycloneAI2` | 1 | 1 | 1 |
| `omegaclone` | `CloneOfBarrierAI2` | 1 | 1 | 1 |
| `mosquaegg` | `MosquaEggAI2` | 1 | 1 | 1 |
| `drakanhealingservant` | `DrakanHealingServantAI2` | 1 | 1 | 1 |
| `klawspawn` | `KlawspawnAI2` | 1 | 1 | 1 |
| `infiltration_rift` | `Infiltration_RiftAI2` | 1 | 1 | 0 |
| `krmagas` | `KromedesMagasAI2` | 1 | 1 | 1 |
| `Q20060` | `GarnonQ20060AI2` | 1 | 1 | 1 |
| `grimreoff` | `GrimreoffAI2` | 1 | 1 | 0 |
| `negarton` | `NegartonAI2` | 1 | 1 | 0 |
| `draidog` | `DraidogAI2` | 1 | 1 | 0 |
| `mercurius` | `MercuriusAI2` | 1 | 1 | 0 |
| `besta` | `BestaAI2` | 1 | 1 | 0 |
| `palgus` | `PalgusAI2` | 1 | 1 | 0 |
| `charlesrunerk` | `CharlesrunerkAI` | 1 | 1 | 0 |
| `edinerk` | `EdinerkAI` | 1 | 1 | 0 |
| `Divine_Bonfire` | `DivineBonfireAI2` | 1 | 1 | 0 |
| `blessed` | `BlessedTotemAI2` | 1 | 1 | 0 |
| `shimmering_spring` | `Shimmering_SpringAI2` | 1 | 1 | 0 |
| `legendary_toy_bear` | `LegendaryToyBearAI2` | 1 | 1 | 0 |
| `AxeSoupBoiler` | `AxeSoupBoilerAI2` | 1 | 1 | 0 |
| `halloween_buff` | `HalloweenBuffCoffinAI2` | 1 | 1 | 0 |
| `f2p_movespeedup` | `MovespeedUpAI2` | 1 | 0 | 0 |
## 4. Дифф-сводка версий

- Только в 5.8: 463 типов (индивидуальные per-NPC/инстансовые AI, свёрнутые в 7.7). Топ-15: `surkana`(56), `defence_bastion`(28), `crafty_esterra_monster`(28), `benoid`(24), `AbbeyPortal`(24), `pvparenarelics`(19), `tombattacker`(18), `crafty_nosra_monster`(16), `empyreanrecordkeeper`(16), `tumon`(15), `recordkeeper`(14), `pashid_assault_pod`(14), `battlefield_trigger_asmodians`(12), `magnorion`(12), `battlefield_trigger_elyos`(12)
- Только в 7.7 (нет в 5.8 и 4.8): 5: `altar_protector`, `examscarecrow`, `f2p_movespeedup`, `holytowerteleport`, `ice_sculptures`
- Только в 4.8: 367 типов (регресс/переименование AL-линии). Топ-20: `simple_abyssguard`(859), `aggressive_no_loot`(393), `base_protector`(253), `fortress_protector`(253), `siege_cannon`(140), `conquest_offering_aggressive`(112), `onedmg_passive`(112), `base_flag`(69), `mercenary`(50), `aggressive_stonespear`(46), `surkana`(42), `eternal_bastion_aggressive`(42), `ahserion_aggressive_npc`(37), `eternal_bastion_mountable`(30), `useSkillAndDie`(28), `modified_iron_wall_aggressive`(27), `conquest_offering_spawner`(24), `onedmg_aggressive`(23), `dredgion_commander`(21), `IDSweep_shugos`(21)

## 5. Handler'ы вне XML (32, 7.7) — движковые/кастомные/легаси

`Haramel_NPCs`, `Shrik`, `betrayericaronix`, `bollvig`, `boques`, `bubblegut`, `celestius`, `daevateleporter`, `emptyaethericcannon`, `emptyetchedcannon`, `enemyservantonedmg`, `fear_dummy`, `fierce_sandmane_tigric`, `guardtower`, `hamerun_the_bleeder`, `heroes_1st_wave_door`, `heroes_2nd_wave_door`, `heroes_3rd_wave_door`, `heroes_4th_wave_door`, `hugeegg`, `kinquid`, `komad_sentry`, `krcorpse`, `nightmare_circus`, `noactionportal`, `omega`, `queenmosqua`, `shifter`, `siege_teleporter`, `simple_abyssguard`, `titanstarturtle`, `ulgornspriggs1`

## 6. Маппинг на NPCSvr64 (наш трек)

1. **Строки AI-типов** (`aggressive`, `trap`, `portal`, `servant`, `siege_protector`, ...) — хендлы для дизasm: искать в NPCSvr64.exe/PDB/ScriptDLL64 (VM) — если строки есть, реестр имён совпадает с эталонным ⇒ карта dispatch по AI-типам готова на 111 значений.
2. Server64-мир: у NC NPC-AI-логика делится Server64↔NPCSvr64; эталоны держат всё в одном GS ⇒ наша разрезка: ai2/ + spawnengine/ → NPCSvr, world/creature → Server64. Спорное (агро-радиус srange, adelay/arate) — по capture R1.
3. Спавны: XML эталонов 7.7 (static_data/spawns/) = готовый референс R3-MVP мира, пока NC-спавны не извлечены.
