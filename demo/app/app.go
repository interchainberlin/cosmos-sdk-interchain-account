package app

import (
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/codec/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	ibcaccount "github.com/chainapsis/cosmos-sdk-interchain-account/x/ibc-account"
	ibcaccountkeeper "github.com/chainapsis/cosmos-sdk-interchain-account/x/ibc-account/keeper"
	ibcaccounttypes "github.com/chainapsis/cosmos-sdk-interchain-account/x/ibc-account/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server/api"
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	transfer "github.com/cosmos/cosmos-sdk/x/ibc-transfer"
	ibctransferkeeper "github.com/cosmos/cosmos-sdk/x/ibc-transfer/keeper"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	ibcclient "github.com/cosmos/cosmos-sdk/x/ibc/02-client"
	ibcclienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	porttypes "github.com/cosmos/cosmos-sdk/x/ibc/05-port/types"
	ibchost "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	ibckeeper "github.com/cosmos/cosmos-sdk/x/ibc/keeper"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const appName = "DemoApp"

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome = os.ExpandEnv("$HOME/.demod")
	DefaultCLIHome  = os.ExpandEnv("$HOME/.democli")

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			paramsclient.ProposalHandler, distrclient.ProposalHandler, upgradeclient.ProposalHandler,
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibcaccount.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName: true,
	}
)

var _ App = (*DemoApp)(nil)

// DemoApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type DemoApp struct {
	*baseapp.BaseApp
	cdc               *codec.Codec
	appCodec          codec.Marshaler
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey
	tkeys   map[string]*sdk.TransientStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// keepers
	accountKeeper    authkeeper.AccountKeeper
	bankKeeper       bankkeeper.Keeper
	capabilityKeeper *capabilitykeeper.Keeper
	stakingKeeper    stakingkeeper.Keeper
	slashingKeeper   slashingkeeper.Keeper
	mintKeeper       mintkeeper.Keeper
	distrKeeper      distrkeeper.Keeper
	govKeeper        govkeeper.Keeper
	crisisKeeper     crisiskeeper.Keeper
	upgradeKeeper    upgradekeeper.Keeper
	paramsKeeper     paramskeeper.Keeper
	ibcKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	evidenceKeeper   evidencekeeper.Keeper
	transferKeeper   ibctransferkeeper.Keeper
	ibcAccountKeeper ibcaccountkeeper.Keeper

	// the module manager
	mm *module.Manager
}

// NewDemoApp returns a reference to an initialized DemoApp.
func NewDemoApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, skipUpgradeHeights map[int64]bool,
	homePath string, invCheckPeriod uint, baseAppOptions ...func(*baseapp.BaseApp),
) *DemoApp {

	// TODO: Remove cdc in favor of appCodec once all modules are migrated.
	encodingConfig := MakeEncodingConfig()
	appCodec := encodingConfig.Marshaler
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(appName, logger, db, authtypes.DefaultTxDecoder(cdc), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetAppVersion(version.Version)
	bApp.GRPCQueryRouter().SetAnyUnpacker(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibchost.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, ibctransfertypes.StoreKey, capabilitytypes.StoreKey,
		ibcaccounttypes.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	app := &DemoApp{
		BaseApp:           bApp,
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	app.paramsKeeper = initParamsKeeper(appCodec, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// set the BaseApp's parameter store
	bApp.SetParamStore(app.paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(std.ConsensusParamsKeyTable()))

	// add capability keeper and ScopeToModule for ibc module
	app.capabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedIBCKeeper := app.capabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedTransferKeeper := app.capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedIBCAccountKeeper := app.capabilityKeeper.ScopeToModule(ibcaccounttypes.ModuleName)

	// add keepers
	app.accountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.GetSubspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, maccPerms,
	)
	app.bankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.accountKeeper, app.GetSubspace(banktypes.ModuleName), app.BlockedAddrs(),
	)
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.accountKeeper, app.bankKeeper, app.GetSubspace(stakingtypes.ModuleName),
	)
	app.mintKeeper = mintkeeper.NewKeeper(
		appCodec, keys[minttypes.StoreKey], app.GetSubspace(minttypes.ModuleName), &stakingKeeper,
		app.accountKeeper, app.bankKeeper, authtypes.FeeCollectorName,
	)
	app.distrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.GetSubspace(distrtypes.ModuleName), app.accountKeeper, app.bankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName, app.ModuleAccountAddrs(),
	)
	app.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, app.GetSubspace(slashingtypes.ModuleName),
	)
	app.crisisKeeper = crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName), invCheckPeriod, app.bankKeeper, authtypes.FeeCollectorName,
	)
	app.upgradeKeeper = upgradekeeper.NewKeeper(skipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec, homePath)

	// register the proposal types
	govRouter := govtypes.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.distrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.upgradeKeeper))
	app.govKeeper = govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.GetSubspace(govtypes.ModuleName), app.accountKeeper, app.bankKeeper,
		&stakingKeeper, govRouter,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.stakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	// Create IBC Keeper
	// TODO: remove amino codec dependency once Tendermint version is upgraded with
	// protobuf changes
	app.ibcKeeper = ibckeeper.NewKeeper(
		app.cdc, appCodec, keys[ibchost.StoreKey], app.stakingKeeper, scopedIBCKeeper,
	)

	// Create Transfer Keepers
	app.transferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey],
		app.ibcKeeper.ChannelKeeper, &app.ibcKeeper.PortKeeper,
		app.accountKeeper, app.bankKeeper, scopedTransferKeeper,
	)
	transferModule := transfer.NewAppModule(app.transferKeeper)

	app.ibcAccountKeeper = ibcaccountkeeper.NewKeeper(appCodec, cdc, keys[ibcaccounttypes.StoreKey],
		map[string]ibcaccountkeeper.CounterpartyInfo{}, app.ibcKeeper.ChannelKeeper, &app.ibcKeeper.PortKeeper,
		app.accountKeeper, scopedIBCAccountKeeper, app.Router(),
	)
	ibcAccountModule := ibcaccount.NewAppModule(app.ibcAccountKeeper)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
	ibcRouter.AddRoute(ibcaccounttypes.ModuleName, ibcAccountModule)
	app.ibcKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], &app.stakingKeeper, app.slashingKeeper,
	)
	evidenceRouter := evidencetypes.NewRouter().
		AddRoute(ibcclienttypes.RouterKey, ibcclient.HandlerClientMisbehaviour(app.ibcKeeper.ClientKeeper))

	evidenceKeeper.SetRouter(evidenceRouter)
	app.evidenceKeeper = *evidenceKeeper

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		genutil.NewAppModule(app.accountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx),
		auth.NewAppModule(appCodec, app.accountKeeper),
		bank.NewAppModule(appCodec, app.bankKeeper, app.accountKeeper),
		capability.NewAppModule(appCodec, *app.capabilityKeeper),
		crisis.NewAppModule(&app.crisisKeeper),
		gov.NewAppModule(appCodec, app.govKeeper, app.accountKeeper, app.bankKeeper),
		mint.NewAppModule(appCodec, app.mintKeeper, app.accountKeeper),
		slashing.NewAppModule(appCodec, app.slashingKeeper, app.accountKeeper, app.bankKeeper, app.stakingKeeper),
		distr.NewAppModule(appCodec, app.distrKeeper, app.accountKeeper, app.bankKeeper, app.stakingKeeper),
		staking.NewAppModule(appCodec, app.stakingKeeper, app.accountKeeper, app.bankKeeper),
		upgrade.NewAppModule(app.upgradeKeeper),
		evidence.NewAppModule(app.evidenceKeeper),
		ibc.NewAppModule(app.ibcKeeper),
		params.NewAppModule(app.paramsKeeper),
		transferModule,
		ibcAccountModule,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName, minttypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		evidencetypes.ModuleName, stakingtypes.ModuleName, ibchost.ModuleName,
	)
	app.mm.SetOrderEndBlockers(crisistypes.ModuleName, govtypes.ModuleName, stakingtypes.ModuleName)

	// NOTE: The genutils moodule must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName, authtypes.ModuleName, distrtypes.ModuleName, stakingtypes.ModuleName, banktypes.ModuleName,
		slashingtypes.ModuleName, govtypes.ModuleName, minttypes.ModuleName, crisistypes.ModuleName,
		ibchost.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName, ibctransfertypes.ModuleName,
	)

	app.mm.RegisterInvariants(&app.crisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter())
	app.mm.RegisterQueryServices(app.GRPCQueryRouter())

	// add test gRPC service for testing gRPC queries in isolation
	testdata.RegisterTestServiceServer(app.GRPCQueryRouter(), testdata.TestServiceImpl{})

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetAnteHandler(
		ante.NewAnteHandler(
			app.accountKeeper, app.bankKeeper, ante.DefaultSigVerificationGasConsumer,
			authtypes.LegacyAminoJSONHandler{},
		),
	)
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	// Initialize and seal the capability keeper so all persistent capabilities
	// are loaded in-memory and prevent any further modules from creating scoped
	// sub-keepers.
	// This must be done during creation of baseapp rather than in InitChain so
	// that in-memory capabilities get regenerated on app restart
	ctx := app.BaseApp.NewUncachedContext(true, abci.Header{})
	app.capabilityKeeper.InitializeAndSeal(ctx)

	return app
}

// MakeCodecs constructs the *std.Codec and *codec.Codec instances used by
// simapp. It is useful for tests and clients who do not want to construct the
// full simapp
func MakeCodecs() (codec.Marshaler, *codec.Codec) {
	config := MakeEncodingConfig()
	return config.Marshaler, config.Amino
}

// Name returns the name of the App
func (app *DemoApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *DemoApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *DemoApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *DemoApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	app.cdc.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	return app.mm.InitGenesis(ctx, app.cdc, genesisState)
}

// LoadHeight loads a particular height
func (app *DemoApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *DemoApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *DemoApp) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// Codec returns DemoApp's codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *DemoApp) Codec() *codec.Codec {
	return app.cdc
}

// AppCodec returns DemoApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *DemoApp) AppCodec() codec.Marshaler {
	return app.appCodec
}

// InterfaceRegistry returns DemoApp's InterfaceRegistry
func (app *DemoApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *DemoApp) GetKey(storeKey string) *sdk.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *DemoApp) GetTKey(storeKey string) *sdk.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *DemoApp) GetMemKey(storeKey string) *sdk.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *DemoApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.paramsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *DemoApp) RegisterAPIRoutes(apiSvr *api.Server) {
	rpc.RegisterRoutes(apiSvr.ClientCtx, apiSvr.Router)
	authrest.RegisterTxRoutes(apiSvr.ClientCtx, apiSvr.Router)
	ModuleBasics.RegisterRESTRoutes(apiSvr.ClientCtx, apiSvr.Router)
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

func MakeEncodingConfig() simparams.EncodingConfig {
	encodingConfig := simparams.MakeEncodingConfig()
	std.RegisterCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaceModules(encodingConfig.InterfaceRegistry)
	return encodingConfig
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.Marshaler, key, tkey sdk.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)

	return paramsKeeper
}