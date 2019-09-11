package constants

//MetaDBDatabasesCollection is the name of the RITA collection
//in the RITA MetaDB that keeps track of RITA managed databases
const MetaDBDatabasesCollection = "databases"

//StrobesCollection contains the name for the RITA freqConn MongoDB collection
const StrobesCollection = "freqConn"

//ConnCollection contains the name for the RITA conn MongoDB collection
const ConnCollection = "conn"

// Version specifies which RITA DB schema the resulting data matches
var Version = "v2.0.0+ActiveCM-IPFIX"

// TODO: Use version in RITA as dep
